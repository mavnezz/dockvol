package encryption

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encryptAll(t *testing.T, plaintext []byte) []byte {
	t.Helper()

	reader, err := GetStreamCipher().EncryptingReader(bytes.NewReader(plaintext))
	require.NoError(t, err)

	ciphertext, err := io.ReadAll(reader)
	require.NoError(t, err)

	return ciphertext
}

func decryptAll(t *testing.T, ciphertext []byte) ([]byte, error) {
	t.Helper()

	reader, err := GetStreamCipher().DecryptingReader(bytes.NewReader(ciphertext))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(reader)
}

func splitStream(t *testing.T, ciphertext []byte) (header []byte, frames [][]byte) {
	t.Helper()

	headerLen := len(streamMagic) + streamSaltSize
	require.Greater(t, len(ciphertext), headerLen)

	header = ciphertext[:headerLen]
	rest := ciphertext[headerLen:]

	for len(rest) > 0 {
		require.GreaterOrEqual(t, len(rest), 4)
		length := int(binary.BigEndian.Uint32(rest[:4]))
		require.GreaterOrEqual(t, len(rest), 4+length)
		frames = append(frames, rest[:4+length])
		rest = rest[4+length:]
	}

	return header, frames
}

func Test_StreamCipher_RoundTrip_VariousSizes(t *testing.T) {
	sizes := []int{
		0,
		1,
		100,
		streamChunkSize - 1,
		streamChunkSize,
		streamChunkSize + 1,
		3 * streamChunkSize,
		3*streamChunkSize + 777,
	}

	for _, size := range sizes {
		plaintext := make([]byte, size)
		_, err := rand.Read(plaintext)
		require.NoError(t, err)

		ciphertext := encryptAll(t, plaintext)
		assert.NotEqual(t, plaintext, ciphertext[len(streamMagic)+streamSaltSize:])

		decrypted, err := decryptAll(t, ciphertext)
		require.NoError(t, err, "size %d", size)
		assert.Equal(t, plaintext, decrypted, "size %d", size)
	}
}

func Test_StreamCipher_SamePlaintext_ProducesDifferentCiphertext(t *testing.T) {
	plaintext := []byte("the same backup content encrypted twice")

	first := encryptAll(t, plaintext)
	second := encryptAll(t, plaintext)

	assert.NotEqual(t, first, second, "a random per-stream salt must make ciphertext differ")

	firstDecrypted, err := decryptAll(t, first)
	require.NoError(t, err)
	secondDecrypted, err := decryptAll(t, second)
	require.NoError(t, err)

	assert.Equal(t, plaintext, firstDecrypted)
	assert.Equal(t, plaintext, secondDecrypted)
}

func Test_DecryptingReader_TamperedChunk_ReturnsError(t *testing.T) {
	ciphertext := encryptAll(t, []byte("authenticated content that must not be silently altered"))

	tampered := bytes.Clone(ciphertext)
	tampered[len(tampered)-1] ^= 0xff

	_, err := decryptAll(t, tampered)
	assert.Error(t, err)
}

func Test_DecryptingReader_TruncatedFinalChunk_ReturnsError(t *testing.T) {
	plaintext := make([]byte, 2*streamChunkSize+10)
	_, err := rand.Read(plaintext)
	require.NoError(t, err)

	ciphertext := encryptAll(t, plaintext)

	header, frames := splitStream(t, ciphertext)
	require.Greater(t, len(frames), 1)

	truncated := bytes.Clone(header)
	for _, frame := range frames[:len(frames)-1] {
		truncated = append(truncated, frame...)
	}

	_, err = decryptAll(t, truncated)
	assert.Error(t, err, "dropping the final chunk must be detected via the end-of-stream flag")
}

func Test_DecryptingReader_ExtendedCiphertext_ReturnsError(t *testing.T) {
	ciphertext := encryptAll(t, []byte("content that is complete on its own"))

	header, frames := splitStream(t, ciphertext)

	extended := bytes.Clone(header)
	for _, frame := range frames {
		extended = append(extended, frame...)
	}
	extended = append(extended, frames[len(frames)-1]...)

	_, err := decryptAll(t, extended)
	assert.Error(t, err, "appending chunks past the end-of-stream flag must be detected")
}

func Test_DecryptingReader_NotAnEncryptedStream_ReturnsError(t *testing.T) {
	notEncrypted := make([]byte, 200)
	_, err := rand.Read(notEncrypted)
	require.NoError(t, err)

	_, err = GetStreamCipher().DecryptingReader(bytes.NewReader(notEncrypted))
	assert.Error(t, err)
}
