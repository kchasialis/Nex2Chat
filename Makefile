export GLOG = warn
export BINLOG = warn
export HTTPLOG = warn
export GORACE = halt_on_error=1
export GNODES = 10
# how many benchmark iterations to average on
export BENCHTIME = "10x"

all: lint vet test

test: test_hw0 test_hw1 test_hw2 test_hw3

test_hw0: test_unit_hw0 test_int_hw0
test_hw1: test_unit_hw1 test_int_hw1
test_hw2: test_unit_hw2 test_int_hw2
test_hw3: test_unit_hw3 test_int_hw3
test_n2c: test_unit_chord test_unit_crypto test_unit_malicious test_unit_social test_unit_utils test_int_n2c

test_unit_hw0:
	go test -timeout 2m -v -race -run Test_HW0 ./peer/tests/unit

test_unit_hw1:
	go test -timeout 2m -v -race -run Test_HW1 ./peer/tests/unit

test_unit_hw2:
	go test -timeout 2m -v -race -run Test_HW2 ./peer/tests/unit

test_unit_hw3:
	go test -timeout 2m -v -race -run Test_HW3 ./peer/tests/unit

test_unit_chord:
	go test -timeout 5m -v -race -run Test_N2C_Chord ./peer/tests/unit

test_unit_crypto:
	go test -timeout 5m -v -race -run Test_N2C_Crypto ./n2ccrypto

test_unit_malicious:
	go test -timeout 5m -v -race -run Test_N2C_Malicious ./peer/malicious

test_unit_social:
	go test -timeout 5m -v -race -run Test_N2C_Social ./peer/tests/unit

test_unit_utils:
	go test -timeout 5m -v -race -run Test_N2C_Utils ./peer/tests/unit

test_int_hw0:
	go test -timeout 5m -v -race -run Test_HW0 ./peer/tests/integration

test_int_hw1:
	go test -timeout 5m -v -race -run Test_HW1 ./peer/tests/integration

test_int_hw2:
	go test -timeout 5m -v -race -run Test_HW2 ./peer/tests/integration

test_int_hw3:
	go test -timeout 5m -v -race -run Test_HW3 ./peer/tests/integration

test_int_n2c:
	@GLOG=no go test -timeout 10m -v -race -run Test_N2C_Integration_No_Malicious ./peer/tests/integration

test_int_n2c_malicious:
	@GLOG=no go test -timeout 10m -v -race -run Test_N2C_Integration_Malicious ./peer/tests/integration

# JSONIFY is set to "-json" in CI to format for GitHub, empty for displaying locally
# || true allows to ignore error code and allow for smoother output logging
test_bench_hw1:
	@GLOG=no go test -v ${JSONIFY} -timeout 15m -run Test_HW1 -v -count 1 --tags=performance -benchtime=${BENCHTIME} ./peer/tests/perf/ || true

test_bench_hw2:
	@GLOG=no go test -v ${JSONIFY} -timeout 21m -run Test_HW2 -v -count 1 --tags=performance -benchtime=3x ./peer/tests/perf/ || true

test_bench_hw3: test_bench_hw3_tlc test_bench_hw3_consensus

test_bench_hw3_tlc:
	@GLOG=no go test -v ${JSONIFY} -timeout 15s -run Test_HW3_BenchmarkTLC -v -count 1 --tags=performance -benchtime=${BENCHTIME} ./peer/tests/perf/ || true

test_bench_hw3_consensus:
	@GLOG=no go test -v ${JSONIFY} -timeout 12m -run Test_HW3_BenchmarkConsensus -v -count 1 --tags=performance -benchtime=${BENCHTIME} ./peer/tests/perf/ || true

lint:
	# Coding style static check.
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.0
	@go mod tidy
	golangci-lint run

vet:
	go vet ./...

