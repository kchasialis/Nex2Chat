package n2ccrypto

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// This function will be called before any tests are run
func TestMain(m *testing.M) {
	// Run the tests
	exitCode := m.Run()

	// Cleanup code
	cleanup()

	// Exit with the exit code from the tests
	os.Exit(exitCode)
}

// Perform cleanup
func cleanup() {
	// Get the current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// File extensions to delete (temporary n2c_crypto keys)
	extensions := []string{".rsa", ".rsa.pub", ".aes"}

	// Iterate over file extensions and delete corresponding files
	for _, ext := range extensions {
		// Find all files with the specified extension in the current directory
		files, err := filepath.Glob(filepath.Join(currentDir, "*"+ext))
		if err != nil {
			panic(err)
		}

		// Delete each file
		for _, file := range files {
			err := os.Remove(file)
			if err != nil {
				panic(err)
			}
		}
	}
}

func Test_N2C_Crypto_GeneratePPKToFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		//// TODO: Add test cases.
		{
			name:    "PK generation to file",
			args:    args{filename: "PK_generation_to_file"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GeneratePPKToFile(tt.args.filename, DefaultBitSize); (err != nil) != tt.wantErr {
				t.Errorf("GeneratePPKToFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_N2C_Crypto_NewPKKModule(t *testing.T) {
	type args struct {
		generatePPK bool
		filename    string
	}
	tests := []struct {
		name    string
		args    args
		want    *PPKModule
		wantErr bool
	}{
		//// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPKKModule(tt.args.generatePPK, tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPKKModule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPKKModule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_N2C_Crypto_LoadPublicKeyToModule(t *testing.T) {
	type args struct {
		module   *PPKModule
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		//// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := LoadPublicKeyToModule(tt.args.module, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("LoadPublicKeyToModule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_N2C_Crypto_LoadPrivateKeyToModule(t *testing.T) {
	type args struct {
		module   *PPKModule
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		//// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadPrivateKeyToModule(tt.args.module, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("LoadPrivateKeyToModule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_N2C_Crypto_LoadPPKToModule(t *testing.T) {
	type args struct {
		module       *PPKModule
		filename     string
		generateKeys bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		//// TODO: Add test cases.
		{
			name: "Load PPK to module",
			args: args{
				module:       (&PPKModule{}),
				filename:     "LOAD_PPK_TEST",
				generateKeys: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.args.generateKeys {
				if err := GeneratePPKToFile(tt.args.filename, DefaultBitSize); (err != nil) != tt.wantErr {
					t.Errorf("GeneratePPKToFile() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if err := LoadPPKToModule(tt.args.module, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("LoadPPKToModule() error = %v, wantErr %v", err, tt.wantErr)
			}

			fmt.Printf("tt.args.module.privateKey: %v\n", tt.args.module.PrivateKey)
			fmt.Printf("tt.args.module.publicKey: %v\n", tt.args.module.PublicKey)
		})
	}
}

func Test_N2C_Crypto_PPKModule_Sign_Verify(t *testing.T) {
	type args struct {
		data        []byte
		encModule   *PPKModule
		decModule   *PPKModule
		encGenerate bool
		encFilename string
		decGenerate bool
		decFilename string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid_Sign_Verify",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Sign_Verify_1",
				decGenerate: false,
				decFilename: "Valid_Sign_Verify_1",
			},
			want:    []byte("test data"),
			wantErr: false,
		},
		{
			name: "Invalid_Sign_Verify",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Sign_Verify_1",
				decGenerate: true,
				decFilename: "Valid_Sign_Verify_2",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			// Initialize the PPKModules
			tt.args.encModule, err = NewPKKModule(tt.args.encGenerate, tt.args.encFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Sign_Verify %s PPK Sign Module Error: %v", tt.name, err)
				return
			}

			tt.args.decModule, err = NewPKKModule(tt.args.decGenerate, tt.args.decFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Sign_Verify %s PPK Verify Module Error: %v", tt.name, err)
				return
			}

			t.Logf("TestPPKModule_Sign_Verify signed text: %v\n", tt.args.data)
			// Sign the data
			signData, err := tt.args.encModule.Sign(tt.args.data, tt.args.encModule.PrivateKey)
			if err != nil {
				t.Errorf("TestPPKModule_Sign_Verify %s Error during signing: %v", tt.name, err)
				return
			}

			// Validate the data
			signedData, err := tt.args.decModule.Verify(tt.args.data, signData, tt.args.decModule.PublicKey)

			if (err != nil && !tt.wantErr) || (err == nil && tt.wantErr) {
				t.Errorf("TestPPKModule_Sign_Verify %s Error during signature verification: %v", tt.name, err)
				return
			}

			if !bytes.Equal(signedData, tt.want) {
				t.Errorf("TestPPKModule_Sign_Verify %s Verify() = %v, want %v", tt.name, signedData, tt.want)
				return
			}
			t.Logf("TestPPKModule_Sign_Verify signed text: %v\n", signedData)
		})
	}
}

func Test_N2C_Crypto_PPKModule_Encrypt_Decrypt(t *testing.T) {
	type args struct {
		data        []byte
		encModule   *PPKModule
		decModule   *PPKModule
		encGenerate bool
		encFilename string
		decGenerate bool
		decFilename string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid_Enc_Dec",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: false,
				decFilename: "Valid_Enc_Dec_1",
			},
			want:    []byte("test data"),
			wantErr: false,
		},
		{
			name: "Invalid_Dec",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: true,
				decFilename: "Valid_Enc_Dec_2",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			// Initialize the PPKModules
			tt.args.encModule, err = NewPKKModule(tt.args.encGenerate, tt.args.encFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Encryption Module Error: %v", tt.name, err)
				return
			}

			tt.args.decModule, err = NewPKKModule(tt.args.decGenerate, tt.args.decFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Decryption Module Error: %v", tt.name, err)
				return
			}

			t.Logf("TestPPKModule_Encrypt_Decrypt encrypted text: %v\n", tt.args.data)
			// Encrypt the data
			encryptedData, err := tt.args.encModule.Encrypt(tt.args.data, tt.args.encModule.PublicKey)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Error during encryption: %v", tt.name, err)
				return
			}

			// Decrypt the data
			decryptedData, err := tt.args.decModule.Decrypt(encryptedData, tt.args.decModule.PrivateKey)

			if (err != nil && !tt.wantErr) || (err == nil && tt.wantErr) {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Error during decryption: %v", tt.name, err)
				return
			}

			if !bytes.Equal(decryptedData, tt.want) {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Decrypt() = %v, want %v", tt.name, decryptedData, tt.want)
				return
			}
			t.Logf("TestPPKModule_Encrypt_Decrypt decrypted text: %v\n", decryptedData)
		})
	}
}

func Test_N2C_Crypto_PPKModule_Encrypt_Decrypt_Encoded_Keys(t *testing.T) {
	type args struct {
		data        []byte
		encModule   *PPKModule
		decModule   *PPKModule
		encGenerate bool
		encFilename string
		decGenerate bool
		decFilename string
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Valid_Enc_Dec",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: false,
				decFilename: "Valid_Enc_Dec_1",
			},
			want:    []byte("test data"),
			wantErr: false,
		},
		{
			name: "Invalid_Dec",
			args: args{
				data:        []byte("test data"),
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: true,
				decFilename: "Valid_Enc_Dec_2",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			// Initialize the PPKModules
			tt.args.encModule, err = NewPKKModule(tt.args.encGenerate, tt.args.encFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Encryption Module Error: %v", tt.name, err)
				return
			}

			tt.args.decModule, err = NewPKKModule(tt.args.decGenerate, tt.args.decFilename)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Decryption Module Error: %v", tt.name, err)
				return
			}

			t.Logf("TestPPKModule_Encrypt_Decrypt encrypted text: %v\n", tt.args.data)
			// Encrypt the data
			encryptedData, err := tt.args.encModule.Encrypt(tt.args.data, tt.args.encModule.PublicKey)
			if err != nil {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Error during encryption: %v", tt.name, err)
				return
			}

			tt.args.decModule.PublicKey, err = DecodePublicKey(GetBytesPublicKey(tt.args.decModule.PublicKey))
			if err != nil {

				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Failed to encode and decode public key: %v", tt.name, err)
				return
			}

			// Decrypt the data
			decryptedData, err := tt.args.decModule.Decrypt(encryptedData, tt.args.decModule.PrivateKey)

			if (err != nil && !tt.wantErr) || (err == nil && tt.wantErr) {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Error during decryption: %v", tt.name, err)
				return
			}

			if !bytes.Equal(decryptedData, tt.want) {
				t.Errorf("TestPPKModule_Encrypt_Decrypt %s Decrypt() = %v, want %v", tt.name, decryptedData, tt.want)
				return
			}
			t.Logf("TestPPKModule_Encrypt_Decrypt decrypted text: %v\n", decryptedData)
		})
	}
}
