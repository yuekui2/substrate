#!/usr/bin/env bash
# Times range reads from a bench node: seed HTTP (Service) vs local dfdaemon proxy.
set -uo pipefail
cd /usr/local/google/home/kuiyue/src/substrate || exit 1
B=benchmarking/image-plugin
ip=$(kubectl -n ate-system get svc nydus-blob-seed -o jsonpath='{.spec.clusterIP}')
pod=$(kubectl -n ate-system get pods -l app=ate-image-plugin,ate.dev/dragonfly-seed=false -o name | head -1)
ref=$(sed -n 2p $B/images-64.jsonl | jq -r .nydus)
repo=${ref%@*}
repo=${repo#us-central1-docker.pkg.dev/}
mapfile -t digests < <(crane manifest "$ref" | jq -r '.layers[].digest' | head -3)
echo "service $ip pod $pod repo $repo"
script=$(cat <<EOF
tok="\$(curl -s -H Metadata-Flavor:Google http://169.254.169.254/computeMetadata/v1/instance/service-accounts/default/token | sed 's/.*"access_token":"\([^"]*\)".*/\1/')"
AUTH="Authorization: Basic \$(printf 'oauth2accesstoken:%s' "\${tok}" | base64 -w0)"
for d in ${digests[*]}; do
  for size in 4096 1048576; do
    for path in seed df; do
      ts=""
      for k in 0 1 2 3 4 5 6 7; do
        off=\$((k * 3 * 1048576))
        if [ \$path = seed ]; then
          t=\$(curl -s -o /dev/null -w '%{http_code}:%{time_total}' -r \$off-\$((off + size - 1)) "http://$ip:4010/v2/$repo/blobs/\$d")
        else
          t=\$(curl -s -o /dev/null -w '%{http_code}:%{time_total}' -r \$off-\$((off + size - 1)) -H "\${AUTH}" -H "X-Dragonfly-Registry: https://us-central1-docker.pkg.dev" "http://127.0.0.1:4001/v2/$repo/blobs/\$d")
        fi
        ts="\$ts \$t"
      done
      echo "\${d:7:12} size=\$size \$path\$ts"
    done
  done
done
EOF
)
kubectl -n ate-system exec "${pod#pod/}" -c nydus-snapshotter -- nsenter -t 1 -m -n -- bash -c "$script"
