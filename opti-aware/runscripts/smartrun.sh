# Copyright (c) 2007-2013 Alysson Bessani, Eduardo Alchieri, Paulo Sousa, and the authors indicated in the @author tags
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

#java -Djava.library.path="./lib:$LD_LIBRARY_PATH:../../../src/main/java/bftsmart/optilog/PrecisionClock" -Djava.security.properties="./config/java.security" -Dlogback.configurationFile="./config/logback.xml" -cp "lib/*" $@

#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "$0")" >/dev/null 2>&1 && pwd)"
LIB_DIR="$SCRIPT_DIR/lib"

# sanity: show we see the ptp lib
[ -f "$LIB_DIR/libptp.so" ] || {
  echo "WARN: $LIB_DIR/libptp.so not found" >&2
}

# also help the dynamic linker
export LD_LIBRARY_PATH="$LIB_DIR:${LD_LIBRARY_PATH:-}"

# debug toggle: DEBUG=1 ./smartrun.sh ...
if [[ "${DEBUG:-0}" != "0" ]]; then
  echo "Using java.library.path=$LIB_DIR"
fi

exec java \
  -Djava.library.path="$LIB_DIR" \
  -Djava.security.properties="$SCRIPT_DIR/config/java.security" \
  -Dlogback.configurationFile="$SCRIPT_DIR/config/logback.xml" \
  -cp "$LIB_DIR/*" \
  "$@"