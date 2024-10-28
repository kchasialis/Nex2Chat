package n2ccrypto

/*
	Public/Private key generation and signature creation/validation module
	using RSA
*/
import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"sync"

	"golang.org/x/xerrors"
)

// good default value should be 2048+, but its reduced to improve performance of prototype showcase
const DefaultBitSize = 2048
const pubKeySuffix = ".rsa.pub"
const privateKeySuffix = ".rsa"

type PPKModule struct {
	PrivateKey       *rsa.PrivateKey
	PublicKey        *rsa.PublicKey
	LoadedKeys       bool
	LoadedPublicKey  bool
	LoadedPrivateKey bool
	moduleLock       sync.RWMutex
	keysize          int
}

// Sign creates a digital signature for the given data using the provided private key.
func (module *PPKModule) Sign(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	// Create a SHA-256 hash of the data
	hashed := crypto.SHA256.New()
	_, err := hashed.Write(data)
	if err != nil {
		return nil, err
	}

	// Sign the hashed data using RSA PKCS#1 v1.5 padding
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed.Sum(nil))
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// Verify checks the validity of a digital signature for the given data using the provided public key.
// returns the data and nil error if the signature is valid
func (module *PPKModule) Verify(data, signature []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Create a SHA-256 hash of the data
	hashed := crypto.SHA256.New()
	_, err := hashed.Write(data)
	if err != nil {
		return nil, err
	}

	// Verify the signature using RSA PKCS#1 v1.5 padding
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed.Sum(nil), signature)
	if err != nil {
		return nil, xerrors.New("signature verification failed")
	}

	return data, nil
}

// Encrypt encrypts the given data using the provided public key.
func (module *PPKModule) Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	// Encrypt the data using RSA PKCS#1 v1.5 padding
	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

// Decrypt decrypts the given data using the provided private key.
func (module *PPKModule) Decrypt(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	// Decrypt the data using RSA PKCS#1 v1.5 padding
	decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, data)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// returns a new PPKModule instance
// has option to either generate new public/private keys or loading them from a file
func NewPKKModule(generatePPK bool, filename string) (*PPKModule, error) {
	return NewPKKModuleVariableKeySize(generatePPK, filename, DefaultBitSize)
}

// returns a new PPKModule instance
// has option to either generate new public/private keys or loading them from a file
func NewPKKModuleVariableKeySize(generatePPK bool, filename string, keysize int) (*PPKModule, error) {
	var module = &PPKModule{}
	var err error

	if filename == "" {
		filename = "key"
	}

	// Generate PPK if asked, try to load it otherwhise
	if generatePPK {
		err = GeneratePPKToFile(filename, keysize)
		if err != nil {
			return nil, err
		}
		module.keysize = keysize
	}

	// load the public and private keys
	err = LoadPPKToModule(module, filename)
	if err != nil {
		return nil, err
	}

	return module, err
}

// Loads the public key into the module
// NOTE: data is only written to module if no error occurs
func LoadPublicKeyToModule(module *PPKModule, filename string) error {
	var err error

	// Load public key
	var pubKeyBytes []byte
	if pubKeyBytes, err = os.ReadFile(filename + pubKeySuffix); err != nil {
		return err
	}

	// decode block
	var blockPtr *pem.Block
	blockPtr, _ = pem.Decode(pubKeyBytes)

	// Parse public key
	var pubKey *rsa.PublicKey
	if pubKey, err = x509.ParsePKCS1PublicKey(blockPtr.Bytes); err != nil {
		return err
	}
	module.moduleLock.Lock()
	module.PublicKey = pubKey
	module.LoadedPublicKey = true
	module.moduleLock.Unlock()

	return err
}

// Loads the private key into the module
// NOTE: data is only written to module if no error occurs
func LoadPrivateKeyToModule(module *PPKModule, filename string) error {
	var err error

	// Load private key
	var privKeyBytes []byte
	if privKeyBytes, err = os.ReadFile(filename + privateKeySuffix); err != nil {
		return err
	}

	// decode block
	var blockPtr *pem.Block
	blockPtr, _ = pem.Decode(privKeyBytes)

	// Parse private key
	var privKey *rsa.PrivateKey
	if privKey, err = x509.ParsePKCS1PrivateKey(blockPtr.Bytes); err != nil {
		return err
	}

	module.moduleLock.Lock()
	module.PrivateKey = privKey
	// Precompute to speed up later operations
	module.PrivateKey.Precompute()
	module.LoadedPrivateKey = true
	module.moduleLock.Unlock()

	return err
}

// Loads the public and private keys into the module
// NOTE: data is only written to module if no error occurs
func LoadPPKToModule(module *PPKModule, filename string) error {
	var err error

	err = LoadPublicKeyToModule(module, filename)
	if err != nil {
		return err
	}

	err = LoadPrivateKeyToModule(module, filename)
	if err != nil {
		return err
	}

	module.moduleLock.Lock()
	module.LoadedKeys = true
	module.moduleLock.Unlock()
	return err
}

func GeneratePPKToFile(filename string, keysize int) error {
	// Generate RSA key
	key, err := rsa.GenerateKey(rand.Reader, keysize)
	if err != nil {
		return err
	}

	// Extract public component.
	pub := key.Public()

	// Encode private key to PKCS#1
	keyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(key),
		},
	)

	// Encode public key to PKCS#1
	pubPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pub.(*rsa.PublicKey)),
		},
	)

	// Write private key to file.
	if err := os.WriteFile(filename+privateKeySuffix, keyPEM, 0600); err != nil {
		return err
	}

	// Write public key to file.
	if err := os.WriteFile(filename+pubKeySuffix, pubPEM, 0600); err != nil {
		return err
	}
	return err
}

// returns a []byte representation of the public key
func GetBytesPublicKey(pubKey *rsa.PublicKey) []byte {
	// Encode the public key to DER format
	derBytes := x509.MarshalPKCS1PublicKey(pubKey)

	// Create a PEM block
	pemBlock := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: derBytes,
	}

	// Encode the PEM block to []byte
	pemBytes := pem.EncodeToMemory(pemBlock)

	return pemBytes
}

// returns a []byte representation of the private key
func GetBytesPrivateKey(privKey *rsa.PrivateKey) []byte {
	// Encode the private key to DER format
	derBytes := x509.MarshalPKCS1PrivateKey(privKey)

	// Create a PEM block
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	}

	// Encode the PEM block to []byte
	pemBytes := pem.EncodeToMemory(pemBlock)

	return pemBytes
}

// DecodePublicKey decodes a PEM-encoded RSA public key from []byte and returns *rsa.PublicKey.
func DecodePublicKey(publicKeyBytes []byte) (*rsa.PublicKey, error) {
	// Decode the PEM block
	pemBlock, _ := pem.Decode(publicKeyBytes)
	if pemBlock == nil {
		return nil, xerrors.Errorf("failed to decode PEM block")
	}

	// Parse the DER-encoded public key
	pubKey, err := x509.ParsePKCS1PublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse public key: %v", err)
	}
	return pubKey, nil
}

// DecodePrivateKey decodes a PEM-encoded RSA private key from []byte and returns *rsa.PrivateKey.
func DecodePrivateKey(privateKeyBytes []byte) (*rsa.PrivateKey, error) {
	// Decode the PEM block
	pemBlock, _ := pem.Decode(privateKeyBytes)
	if pemBlock == nil {
		return nil, xerrors.Errorf("failed to decode PEM block")
	}

	// Parse the DER-encoded private key
	privKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse private key: %v", err)
	}

	return privKey, nil
}

func (module *PPKModule) GetPrivateKey() *rsa.PrivateKey {

	module.moduleLock.Lock()
	var ptrKey = module.PrivateKey
	module.moduleLock.Unlock()
	return ptrKey
}

func (module *PPKModule) GetPublicKey() *rsa.PublicKey {

	module.moduleLock.Lock()
	var ptrKey = module.PublicKey
	module.moduleLock.Unlock()
	return ptrKey
}
