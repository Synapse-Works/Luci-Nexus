#!/bin/sh
# LuciNexus Universal OpenWrt Installer
# Usage: wget -qO- https://raw.githubusercontent.com/lucinexus/lucinexus/main/install.sh | sh

set -e

REPO="lucinexus/lucinexus"
VERSION="2.1.0"
BIN_DIR="/usr/bin"
CONFIG_DIR="/etc/config"
INIT_DIR="/etc/init.d"

echo "=========================================================="
echo "  🚀 Installing LuciNexus v${VERSION} on OpenWrt..."
echo "=========================================================="

ARCH=$(uname -m)
case "$ARCH" in
    mips*)
        if [ "$(echo -n I | hexdump -o | awk '{ print substr($2,6,1); exit}')" = "1" ]; then
            TARGET="mipsel"
        else
            TARGET="mips"
        fi
        ;;
    armv7*|armv6*)
        TARGET="arm"
        ;;
    aarch64*|arm64*)
        TARGET="arm64"
        ;;
    x86_64)
        TARGET="amd64"
        ;;
    *)
        echo "❌ Unsupported CPU architecture: $ARCH"
        exit 1
        ;;
esac

echo "✓ Detected architecture: $ARCH -> target binary: lucinexus-$TARGET"

echo "✓ Checking dependencies..."
opkg update >/dev/null 2>&1 || true
for pkg in ca-certificates libustream-mbedtls curl; do
    if ! opkg list-installed | grep -q "^$pkg"; then
        echo "  -> Installing $pkg..."
        opkg install "$pkg" >/dev/null 2>&1 || true
    fi
done

URL="https://github.com/${REPO}/releases/download/v${VERSION}/lucinexus-${TARGET}"
echo "✓ Downloading binary from $URL..."
wget -q -O "${BIN_DIR}/lucinexus" "$URL" || {
    echo "❌ Failed to download binary. Please verify internet connection."
    exit 1
}
chmod +x "${BIN_DIR}/lucinexus"

if [ ! -f "${CONFIG_DIR}/lucinexus" ]; then
    echo "✓ Creating default configuration at ${CONFIG_DIR}/lucinexus..."
    cat << 'CONFIG_EOF' > "${CONFIG_DIR}/lucinexus"
config lucinexus 'global'
	option enabled '1'
	option port '9999'
	option dry_run '1'
	option auto_rollback '1'
	option enable_fastpath '1'
	option sparse_hydration '1'

config api 'api'
	option provider 'gemini'
	option model 'gemini-2.5-flash'
	option key ''
CONFIG_EOF
fi

echo "✓ Registering Procd init script..."
cat << 'INIT_EOF' > "${INIT_DIR}/lucinexus"
#!/bin/sh /etc/rc.common

USE_PROCD=1
START=95
STOP=10

PROG=/usr/bin/lucinexus

start_service() {
	config_load lucinexus
	local enabled port
	config_get_bool enabled global enabled 0
	config_get port global port 9999

	[ "$enabled" -eq 1 ] || return 0

	procd_open_instance
	procd_set_param command "$PROG" -server
	procd_set_param respawn 3600 5 5
	procd_set_param stdout 1
	procd_set_param stderr 1
	procd_set_param limits core="0:0"
	procd_close_instance
}

stop_service() {
	rm -rf /tmp/lucinexus/snapshots/* 2>/dev/null || true
}
INIT_EOF
chmod +x "${INIT_DIR}/lucinexus"
"${INIT_DIR}/lucinexus" enable
"${INIT_DIR}/lucinexus" restart

echo "=========================================================="
echo "  🎉 LuciNexus v${VERSION} installed successfully!"
echo "  Run 'lucinexus -version' to test."
echo "  Configure your API key with:"
echo "    uci set lucinexus.@api[0].key='YOUR_API_KEY'"
echo "    uci commit lucinexus"
echo "  Or configure a custom provider via cURL:"
echo "    lucinexus config --curl '<curl_command>'"
echo "=========================================================="
