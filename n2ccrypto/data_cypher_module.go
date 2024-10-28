package n2ccrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"sync"
)

const DefaultKeySize = 32 // AES-256 key size in bytes, either 16,24 or 32
const DCypherModuleExtension = ".aes"

type DCypherModule struct {
	Key        []byte
	LoadedKey  bool
	moduleLock sync.RWMutex
}

func NewDCypherModule(generateKey bool, filename string) (*DCypherModule, error) {
	var module = &DCypherModule{}
	var err error

	if filename == "" {
		filename = "key"
	}

	if generateKey {
		err = GenerateKeyToFile(filename)
		if err != nil {
			return nil, err
		}
	}

	err = LoadKeyFileToModule(module, filename)
	if err != nil {
		return nil, err
	}

	return module, nil
}

func (module *DCypherModule) Encrypt(data []byte) ([]byte, error) {
	if !module.LoadedKey {
		return nil, errors.New("key not loaded")
	}
	module.moduleLock.Lock()
	block, err := aes.NewCipher(module.Key)
	module.moduleLock.Unlock()
	if err != nil {
		return nil, err
	}

	// data = pkcs7Pad(data, aes.BlockSize)
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)

	return ciphertext, nil
}

func (module *DCypherModule) Decrypt(data []byte) ([]byte, error) {
	if !module.LoadedKey {
		return nil, errors.New("key not loaded")
	}

	module.moduleLock.Lock()
	block, err := aes.NewCipher(module.Key)
	module.moduleLock.Unlock()
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]

	mode := cipher.NewCFBDecrypter(block, iv)
	mode.XORKeyStream(data, data)
	// data, err = pkcs7Unpad(data)

	return data, nil
}

func (module *DCypherModule) LoadKeyToModule(keyBytes []byte) {

	module.moduleLock.Lock()
	module.Key = keyBytes
	module.LoadedKey = true
	module.moduleLock.Unlock()

}

func LoadKeyFileToModule(module *DCypherModule, filename string) error {
	keyBytes, err := os.ReadFile(filename + DCypherModuleExtension)
	if err != nil {
		return err
	}
	module.LoadKeyToModule(keyBytes)

	return nil
}

func GenerateKeyToFile(filename string) error {
	key := make([]byte, DefaultKeySize)

	// rand.Read populates key with random data
	if _, err := rand.Read(key); err != nil {
		return err
	}

	err := os.WriteFile(filename+DCypherModuleExtension, key, 0600)
	return err
}

func (module *DCypherModule) GetKey() []byte {

	module.moduleLock.Lock()
	var key = module.Key
	module.moduleLock.Unlock()
	return key
}
