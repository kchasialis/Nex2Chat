package n2ccrypto

import (
	"bytes"
	"testing"
)

func Test_N2C_Crypto_HashWithConfig(t *testing.T) {
	type args struct {
		data       []byte
		salt       []byte
		timeCost   uint32
		memoryCost uint32
		threads    uint8
		keyLen     uint32
	}

	tests := []struct {
		name    string
		wants   []byte
		wantErr bool
		args    args
	}{
		{
			name: "Different hash outputs for different time and memory costs",
			args: args{
				data:       []byte("testdata"),
				salt:       []byte("somesalt"),
				timeCost:   1,
				memoryCost: 64 * 1024,
				threads:    DefaultThreads,
				keyLen:     DefaultKeyLen,
			},
			wants:   StrongHashWithConfig([]byte("testdata"), []byte("somesalt"), 2, 32*1024, DefaultThreads, DefaultKeyLen, false),
			wantErr: true,
		},
		{
			name: "Same data and salt hashed twice has the same resulting hash",
			args: args{
				data:       []byte("testdata"),
				salt:       []byte("somesalt"),
				timeCost:   DefaultTimeCost,
				memoryCost: DefaultMemoryCost,
				threads:    DefaultThreads,
				keyLen:     DefaultKeyLen,
			},
			wants:   StrongHashWithConfig([]byte("testdata"), []byte("somesalt"), DefaultTimeCost, DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false),
			wantErr: false,
		},
		{
			name: "Data with no salt has a different hash output from data with salt",
			args: args{
				data:       []byte("testdata"),
				salt:       nil,
				timeCost:   DefaultTimeCost,
				memoryCost: DefaultMemoryCost,
				threads:    DefaultThreads,
				keyLen:     DefaultKeyLen,
			},
			wants:   StrongHashWithConfig([]byte("testdata"), []byte("somesalt"), DefaultTimeCost, DefaultMemoryCost, DefaultThreads, DefaultKeyLen, false),
			wantErr: true,
		},
		{
			name: "Data with no salt has a different hash output from data with salt",
			args: args{
				data:       []byte("testdata"),
				salt:       nil,
				timeCost:   DefaultTimeCost,
				memoryCost: DefaultMemoryCost,
				threads:    DefaultThreads,
				keyLen:     DefaultKeyLen,
			},
			wants:   ShaHashSalted([]byte("testdata"), []byte("somesalt")),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := StrongHashWithConfig(tt.args.data, tt.args.salt, tt.args.timeCost, tt.args.memoryCost, tt.args.threads, tt.args.keyLen, false)

			// Compare the obtained hash with the expected hash for each test case
			if !bytes.Equal(hash, tt.wants) && !tt.wantErr {
				t.Errorf("HashWithConfig() got = %v, want %v", hash, tt.wants)
			}
		})
	}
}
