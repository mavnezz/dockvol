package encryption

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"dockvol-backend/internal/features/encryption/secrets"
)

// StreamCipher encrypts and decrypts arbitrarily large streams (backup tarballs)
// with chunked AES-256-GCM. Each stream derives its own key from the master key
// via HKDF with a random salt, so counter-based per-chunk nonces never repeat
// across streams. Every chunk authenticates its index and an end-of-stream flag
// as additional data, which makes truncation, reordering, and extension of the
// ciphertext detectable on decrypt.
type StreamCipher struct {
	secretKeyService *secrets.SecretKeyService
}

const (
	streamMagic     = "DVENC1"
	streamKeyInfo   = "dockvol-backup-content"
	streamSaltSize  = 32
	streamChunkSize = 64 * 1024
)

// streamMaxCipherChunk caps a length-prefixed ciphertext chunk so a corrupt or
// hostile length field can't drive an unbounded allocation. GCM adds at most its
// overhead on top of one plaintext chunk.
const streamMaxCipherChunk = streamChunkSize + 64

func (c *StreamCipher) EncryptingReader(source io.Reader) (io.Reader, error) {
	salt := make([]byte, streamSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate stream salt: %w", err)
	}

	gcm, err := c.newGCM(salt)
	if err != nil {
		return nil, err
	}

	header := append([]byte(streamMagic), salt...)

	return &encryptingReader{
		source:   source,
		gcm:      gcm,
		plainBuf: make([]byte, streamChunkSize),
		out:      bytes.NewBuffer(header),
	}, nil
}

func (c *StreamCipher) DecryptingReader(source io.Reader) (io.Reader, error) {
	buffered := bufio.NewReader(source)

	magic := make([]byte, len(streamMagic))
	if _, err := io.ReadFull(buffered, magic); err != nil {
		return nil, fmt.Errorf("read stream header: %w", err)
	}

	if string(magic) != streamMagic {
		return nil, errors.New("not an encrypted backup stream")
	}

	salt := make([]byte, streamSaltSize)
	if _, err := io.ReadFull(buffered, salt); err != nil {
		return nil, fmt.Errorf("read stream salt: %w", err)
	}

	gcm, err := c.newGCM(salt)
	if err != nil {
		return nil, err
	}

	return &decryptingReader{source: buffered, gcm: gcm, out: &bytes.Buffer{}}, nil
}

func (c *StreamCipher) newGCM(salt []byte) (cipher.AEAD, error) {
	masterKey, err := c.secretKeyService.GetSecretKey()
	if err != nil {
		return nil, fmt.Errorf("get master key: %w", err)
	}

	key, err := hkdf.Key(sha256.New, []byte(masterKey), salt, streamKeyInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("derive stream key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	return cipher.NewGCM(block)
}

func chunkNonceAndAAD(gcm cipher.AEAD, index uint64, isLast bool) (nonce, aad []byte) {
	nonce = make([]byte, gcm.NonceSize())
	binary.BigEndian.PutUint64(nonce[gcm.NonceSize()-8:], index)

	aad = make([]byte, 9)
	binary.BigEndian.PutUint64(aad[:8], index)
	if isLast {
		aad[8] = 1
	}

	return nonce, aad
}

type encryptingReader struct {
	source   io.Reader
	gcm      cipher.AEAD
	plainBuf []byte
	out      *bytes.Buffer
	index    uint64
	done     bool
}

func (r *encryptingReader) Read(p []byte) (int, error) {
	for r.out.Len() == 0 && !r.done {
		if err := r.sealNextChunk(); err != nil {
			return 0, err
		}
	}

	return r.out.Read(p)
}

func (r *encryptingReader) sealNextChunk() error {
	read, err := io.ReadFull(r.source, r.plainBuf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("read plaintext chunk: %w", err)
	}

	isLast := errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)

	nonce, aad := chunkNonceAndAAD(r.gcm, r.index, isLast)
	ciphertext := r.gcm.Seal(nil, nonce, r.plainBuf[:read], aad)

	lengthPrefix := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthPrefix, uint32(len(ciphertext)))

	r.out.Write(lengthPrefix)
	r.out.Write(ciphertext)

	r.index++
	r.done = isLast

	return nil
}

type decryptingReader struct {
	source  *bufio.Reader
	gcm     cipher.AEAD
	out     *bytes.Buffer
	index   uint64
	pending []byte
	primed  bool
	done    bool
}

func (r *decryptingReader) Read(p []byte) (int, error) {
	for r.out.Len() == 0 && !r.done {
		if err := r.openNextChunk(); err != nil {
			return 0, err
		}
	}

	return r.out.Read(p)
}

func (r *decryptingReader) openNextChunk() error {
	if !r.primed {
		first, hasChunk, err := r.readChunk()
		if err != nil {
			return err
		}

		if !hasChunk {
			return errors.New("encrypted backup is empty or truncated")
		}

		r.pending = first
		r.primed = true
	}

	next, hasNext, err := r.readChunk()
	if err != nil {
		return err
	}

	isLast := !hasNext
	if err := r.openPending(isLast); err != nil {
		return err
	}

	if isLast {
		r.done = true

		return nil
	}

	r.pending = next

	return nil
}

func (r *decryptingReader) openPending(isLast bool) error {
	nonce, aad := chunkNonceAndAAD(r.gcm, r.index, isLast)

	plaintext, err := r.gcm.Open(nil, nonce, r.pending, aad)
	if err != nil {
		return fmt.Errorf("decrypt chunk: %w", err)
	}

	r.out.Write(plaintext)
	r.index++

	return nil
}

func (r *decryptingReader) readChunk() (chunk []byte, hasChunk bool, err error) {
	lengthPrefix := make([]byte, 4)

	_, err = io.ReadFull(r.source, lengthPrefix)
	if errors.Is(err, io.EOF) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("read chunk length: %w", err)
	}

	length := binary.BigEndian.Uint32(lengthPrefix)
	if length == 0 || length > streamMaxCipherChunk {
		return nil, false, fmt.Errorf("invalid encrypted chunk length: %d", length)
	}

	ciphertext := make([]byte, length)
	if _, err := io.ReadFull(r.source, ciphertext); err != nil {
		return nil, false, fmt.Errorf("read chunk body: %w", err)
	}

	return ciphertext, true, nil
}
