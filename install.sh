#!/bin/sh
# LuciNexus Universal OpenWrt Installer
# Usage: wget -qO- --no-check-certificate https://raw.githubusercontent.com/Synapse-Works/Luci-Nexus/main/install.sh | sh
#   or:  curl -sSL https://raw.githubusercontent.com/Synapse-Works/Luci-Nexus/main/install.sh | sh

set -e

REPO="Synapse-Works/Luci-Nexus"
VERSION="1.0.0"
BIN_DIR="${BIN_DIR:-/usr/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/config}"
INIT_DIR="${INIT_DIR:-/etc/init.d}"
LUCI_CONTROLLER_DIR="${LUCI_CONTROLLER_DIR:-/usr/lib/lua/luci/controller}"
LUCI_VIEW_DIR="${LUCI_VIEW_DIR:-/www/luci-static/resources/view/lucinexus}"

echo "=========================================================="
echo "  🚀 Installing LuciNexus v${VERSION} on OpenWrt..."
echo "=========================================================="

ARCH=$(uname -m)
case "$ARCH" in
    mips*)
        if [ -f /etc/openwrt_release ] && grep -qi 'mipsel' /etc/openwrt_release; then
            TARGET="mipsel"
        elif echo -n I | hexdump -o 2>/dev/null | awk '{ print substr($2,6,1); exit}' | grep -q "1"; then
            TARGET="mipsel"
        elif echo -n I | od -to2 2>/dev/null | awk '{ print substr($2,6,1); exit}' | grep -q "1"; then
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
if command -v apk >/dev/null 2>&1; then
    apk update >/dev/null 2>&1 || true
    for pkg in ca-certificates libustream-mbedtls curl; do
        if ! apk info 2>/dev/null | grep -q "^$pkg"; then
            echo "  -> Installing $pkg via apk..."
            apk add "$pkg" >/dev/null 2>&1 || true
        fi
    done
elif command -v opkg >/dev/null 2>&1; then
    opkg update >/dev/null 2>&1 || true
    for pkg in ca-certificates libustream-mbedtls curl; do
        if ! opkg list-installed 2>/dev/null | grep -q "^$pkg"; then
            echo "  -> Installing $pkg via opkg..."
            opkg install "$pkg" >/dev/null 2>&1 || true
        fi
    done
fi

mkdir -p "${BIN_DIR}" "${CONFIG_DIR}" "${INIT_DIR}"

URL="https://github.com/${REPO}/releases/download/v${VERSION}/lucinexus-${TARGET}"

TMP_BIN="/tmp/lucinexus.$$"
rm -f "$TMP_BIN"

if [ -f "./bin/lucinexus-${TARGET}" ]; then
    echo "✓ Found local binary ./bin/lucinexus-${TARGET}, installing to ${BIN_DIR}/lucinexus..."
    cp "./bin/lucinexus-${TARGET}" "${BIN_DIR}/lucinexus"
elif [ -f "./bin/lucinexus" ]; then
    echo "✓ Found local binary ./bin/lucinexus, installing to ${BIN_DIR}/lucinexus..."
    cp "./bin/lucinexus" "${BIN_DIR}/lucinexus"
else
    echo "✓ Downloading binary for $TARGET from $URL..."
    DL_OK=0
    if command -v curl >/dev/null 2>&1; then
        if curl -fsSL -o "$TMP_BIN" "$URL" 2>/dev/null; then
            DL_OK=1
        fi
    fi
    if [ "$DL_OK" -eq 0 ]; then
        if wget -q --no-check-certificate -O "$TMP_BIN" "$URL" 2>/dev/null || wget -q -O "$TMP_BIN" "$URL" 2>/dev/null; then
            if [ -s "$TMP_BIN" ] && ! grep -qi "Not Found" "$TMP_BIN" 2>/dev/null; then
                DL_OK=1
            fi
        fi
    fi

    FILE_SIZE=0
    if [ -f "$TMP_BIN" ]; then
        FILE_SIZE=$(wc -c < "$TMP_BIN" 2>/dev/null || echo 0)
    fi

    if [ "$DL_OK" -eq 1 ] && [ "$FILE_SIZE" -gt 100000 ]; then
        mv "$TMP_BIN" "${BIN_DIR}/lucinexus"
        echo "✓ Successfully installed binary ($((FILE_SIZE / 1024)) KB)."
    else
        rm -f "$TMP_BIN"
        echo "❌ Binary download failed: Release asset 'lucinexus-${TARGET}' is not available on GitHub Releases yet."
        echo ""
        ROUTER_IP=$(uci get network.lan.ipaddr 2>/dev/null | cut -d/ -f1)
        ROUTER_IP="${ROUTER_IP:-<router-ip>}"
        echo "  To fix this immediately, transfer the pre-compiled binary from your computer:"
        echo "    scp bin/lucinexus-${TARGET} root@${ROUTER_IP}:/usr/bin/lucinexus"
        echo "    chmod +x /usr/bin/lucinexus"
        exit 1
    fi
fi

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

if [ -f /etc/rc.common ]; then
    "${INIT_DIR}/lucinexus" enable 2>/dev/null || true
    "${INIT_DIR}/lucinexus" restart 2>/dev/null || true
fi

# Install LuCI Web UI components if LuCI directory or local package files exist
if [ -d "/usr/lib/lua/luci" ] || [ -f "./package/luci-app-lucinexus/luasrc/controller/lucinexus.lua" ]; then
    echo "✓ Installing LuCI web interface files..."
    mkdir -p "$LUCI_CONTROLLER_DIR" "$LUCI_VIEW_DIR"
    if [ -f "./package/luci-app-lucinexus/luasrc/controller/lucinexus.lua" ]; then
        cp "./package/luci-app-lucinexus/luasrc/controller/lucinexus.lua" "${LUCI_CONTROLLER_DIR}/lucinexus.lua"
        cp "./package/luci-app-lucinexus/htdocs/luci-static/resources/view/lucinexus/app.js" "${LUCI_VIEW_DIR}/app.js"
    else
        RAW_URL="https://raw.githubusercontent.com/${REPO}/main/package/luci-app-lucinexus"
        if command -v curl >/dev/null 2>&1; then
            curl -sSL -o "${LUCI_CONTROLLER_DIR}/lucinexus.lua" "${RAW_URL}/luasrc/controller/lucinexus.lua" 2>/dev/null || true
            curl -sSL -o "${LUCI_VIEW_DIR}/app.js" "${RAW_URL}/htdocs/luci-static/resources/view/lucinexus/app.js" 2>/dev/null || true
        fi
    fi
    rm -rf /tmp/luci-indexcache /tmp/luci-modulecache 2>/dev/null || true
fi

echo "=========================================================="
echo "  🎉 LuciNexus v${VERSION} installed successfully!"
echo "  Run 'lucinexus -version' to test."
echo "  Configure your API key with:"
echo "    uci set lucinexus.@api[0].key='YOUR_API_KEY'"
echo "    uci commit lucinexus"
echo "  Or configure a custom provider via cURL:"
echo "    lucinexus -curl '<curl_command>'"
echo "=========================================================="
