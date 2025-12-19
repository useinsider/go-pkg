#!/bin/bash

declare -A EXPECTED=(
    ["github.com/DATA-DOG/go-sqlmock"]="v1.5.0"
    ["github.com/Jamil-Najafov/go-aws-ssm"]="v0.9.0"
    ["github.com/aws/aws-sdk-go"]="v1.44.3"
    ["github.com/aws/aws-sdk-go-v2"]="v1.23.1"
    ["github.com/aws/aws-sdk-go-v2/config"]="v1.25.4"
    ["github.com/aws/aws-sdk-go-v2/service/sqs"]="v1.28.2"
    ["github.com/aws/smithy-go"]="v1.17.0"
    ["github.com/getsentry/sentry-go"]="v0.13.0"
    ["github.com/go-redis/redis"]="v6.15.9+incompatible"
    ["github.com/golang/mock"]="v1.6.0"
    ["github.com/google/uuid"]="v1.3.1"
    ["github.com/jellydator/ttlcache/v3"]="v3.0.0"
    ["github.com/pkg/errors"]="v0.9.1"
    ["github.com/slok/goresilience"]="v0.2.0"
    ["github.com/stretchr/testify"]="v1.8.1"
    ["go.uber.org/mock"]="v0.3.0"
    ["go.uber.org/zap"]="v1.26.0"
    ["gorm.io/driver/mysql"]="v1.3.4"
    ["gorm.io/gorm"]="v1.23.7"
)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

errors=0

for dir in "$ROOT_DIR"/ins*/; do
    module=$(basename "$dir")
    gomod="$dir/go.mod"

    [[ ! -f "$gomod" ]] && continue

    while IFS= read -r line; do
        # Extract package and version from require lines
        if [[ $line =~ ^[[:space:]]+(github\.com|go\.|gorm\.io)[^[:space:]]+[[:space:]]+(v[^[:space:]]+) ]]; then
            pkg=$(echo "$line" | awk '{print $1}')
            ver=$(echo "$line" | awk '{print $2}')

            # Skip indirect dependencies
            [[ $line == *"// indirect"* ]] && continue

            # Check if we have an expected version for this package
            if [[ -n "${EXPECTED[$pkg]}" ]]; then
                expected="${EXPECTED[$pkg]}"
                if [[ "$ver" != "$expected" ]]; then
                    echo -e "${RED}MISMATCH${NC} $module: $pkg"
                    echo "  expected: $expected"
                    echo "  actual:   $ver"
                    ((errors++))
                fi
            fi
        fi
    done < "$gomod"
done

if [[ $errors -eq 0 ]]; then
    echo -e "${GREEN}All dependencies match expected versions${NC}"
    exit 0
else
    echo -e "\n${RED}Found $errors mismatches${NC}"
    exit 1
fi
