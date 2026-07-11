package encryption

import "dockvol-backend/internal/features/encryption/secrets"

var fieldEncryptor = &SecretKeyFieldEncryptor{
	secrets.GetSecretKeyService(),
}

var streamCipher = &StreamCipher{
	secrets.GetSecretKeyService(),
}

func GetFieldEncryptor() FieldEncryptor {
	return fieldEncryptor
}

func GetStreamCipher() *StreamCipher {
	return streamCipher
}
