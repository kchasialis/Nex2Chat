package n2ccrypto

import (
	"bytes"
	"crypto/rand"
	"reflect"
	"testing"
)

func Test_N2C_Crypto_DCM_GenerateKeyToFile(t *testing.T) {
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
			if err := GenerateKeyToFile(tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("GenerateKeyToFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_N2C_Crypto_DCM_NewDCypherModule(t *testing.T) {
	type args struct {
		generatePPK bool
		filename    string
	}
	tests := []struct {
		name    string
		args    args
		want    *DCypherModule
		wantErr bool
	}{
		//// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDCypherModule(tt.args.generatePPK, tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDCypherModule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDCypherModule() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_N2C_Crypto_DCM_LoadKeyToModule(t *testing.T) {
	type args struct {
		module   *DCypherModule
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

			if err := LoadKeyFileToModule(tt.args.module, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("LoadPublicKeyToModule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_N2C_Crypto_DCM_DCypherModule_Encrypt_Decrypt(t *testing.T) {
	randData := make([]byte, 10000)
	// rand.Read populates key with random data
	if _, err := rand.Read(randData); err != nil {
		t.Error(err.Error())
	}
	// randData = []byte(strings.Repeat("Test data,", 100))
	type args struct {
		data        []byte
		encModule   *DCypherModule
		decModule   *DCypherModule
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
				data:        randData,
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: false,
				decFilename: "Valid_Enc_Dec_1",
			},
			want:    randData,
			wantErr: false,
		},
		{
			name: "Invalid_Dec",
			args: args{
				data:        randData,
				encGenerate: true,
				encFilename: "Valid_Enc_Dec_1",
				decGenerate: true,
				decFilename: "Valid_Enc_Dec_2",
			},
			want:    []byte("not equal"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			// Initialize the DCypherModules
			tt.args.encModule, err = NewDCypherModule(tt.args.encGenerate, tt.args.encFilename)
			if err != nil {
				t.Errorf("TestDCypherModule_Encrypt_Decrypt %s Encryption Module Error: %v", tt.name, err)
				return
			}

			tt.args.decModule, err = NewDCypherModule(tt.args.decGenerate, tt.args.decFilename)
			if err != nil {
				t.Errorf("TestDCypherModule_Encrypt_Decrypt %s Decryption Module Error: %v", tt.name, err)
				return
			}

			// t.Logf("TestDCypherModule_Encrypt_Decrypt encrypted text: %v\n", tt.args.data)
			// Encrypt the data
			encryptedData, err := tt.args.encModule.Encrypt(tt.args.data)
			if err != nil {
				t.Errorf("TestDCypherModule_Encrypt_Decrypt %s Error during encryption: %v", tt.name, err)
				return
			}

			// Decrypt the data
			decryptedData, err := tt.args.decModule.Decrypt(encryptedData)

			var bytesAreEqual = bytes.Equal(decryptedData, tt.want)

			// t.Logf("TestDCypherModule_Encrypt_Decrypt original text: %v\n", string(tt.args.data))
			// t.Logf("TestDCypherModule_Encrypt_Decrypt decrypted text: %v\n", string(decryptedData))

			if !bytesAreEqual && !tt.wantErr {
				t.Errorf("TestDCypherModule_Encrypt_Decrypt %s Decrypt() = %v, want %v", tt.name, decryptedData, tt.want)
				return
			}
			// t.Logf("TestDCypherModule_Encrypt_Decrypt decrypted text: %v\n", decryptedData)
		})
	}
}
