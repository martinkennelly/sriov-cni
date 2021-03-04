#!/bin/bash
set -eo pipefail
# Iterate over SRIOV CNI fuzzer config files located test/fuzz/configs
# Alter the iteration for each config file by setting env var TEST_ITERATIONS
# Alter the path of SRIOV binary by setting env var SRIOV_PATH
# Log of each fuzzer run for each config file will be output to STDOUT and file
app_name="sriov-fuzzy"
root="$(cd "$(dirname "$0")/.."; printf "%s" "$(pwd)")"
fuzz="${root}/test/fuzz"
fuzz_config="${fuzz}/configs"
test_iter="${TEST_ITERATIONS:=10000}"
sriov="${SRIOV_PATH:=${root}/build/sriov}"
radamsa="${RADAMSA_PATH:=/usr/bin/radamsa}"

# Check if SRIOV CNI & go available
check_requirements() {
  echo "# checking requirements"
  if [ ! -f "${sriov}" ]; then
    echo "SRIOV CNI is not at path '${sriov}'"
    exit 1
  fi

  if ! command -v go &> /dev/null; then
    echo "## go is not available"
    exit 1
  fi
  mkdir -p "${root}/build"
}

build_fuzzer() {
  if [ ! -f "${root}/build/${app_name}" ]; then
    echo "# building SRIOV CNI fuzzer"
    go build -ldflags "-s -w" -buildmode=pie -o "${root}/build/${app_name}" "${fuzz}/sriov-fuzzy.go"
  else
    echo "# SRIOV CNI already built"
  fi
}

check_requirements
build_fuzzer
echo "# executing SRIOV CNI fuzzer '${test_iter}' time(s) for all configs in '${fuzz_config}'"
for config in ${fuzz_config}/*; do
  log_dir="$(mktemp -q -p '/tmp' -d -t sriov-cni-XXXXXX)"
  log_file="$(mktemp -q "$log_dir/sriov-cni-fuzzXXXX.log")"
  bash -c "${root}/build/${app_name} --tests ${test_iter} --radamsa ${radamsa} --cni ${sriov} --config ${config} --panicOnly --out ${log_file}"
  echo "## Config file - '${config}' ##" && cat "${config}"
  echo "## Log file ## - '${log_file}'" && cat "${log_file}"
done
