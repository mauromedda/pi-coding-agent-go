#!/bin/bash
# ABOUTME: Installer script for the pi-go binary from GitHub Releases
# ABOUTME: Supports platform auto-detection, checksum verification, offline mode, and piped execution

set -euo pipefail

# ============================================================================
# GLOBAL VARIABLES AND CONSTANTS
# ============================================================================
readonly SCRIPT_NAME="install.sh"
readonly GITHUB_REPO="mauromedda/pi-coding-agent-go"
readonly BINARY_NAME="pi-go"
readonly GITHUB_API="https://api.github.com/repos/${GITHUB_REPO}"
readonly GITHUB_RELEASES="https://github.com/${GITHUB_REPO}/releases"

# Exit codes
declare -ri EXIT_SUCCESS=0
declare -ri EXIT_FAILURE=1
declare -ri EXIT_USAGE=2

# Default configuration
declare INSTALL_DIR="${HOME}/.local/bin"
declare INSTALL_SYSTEM=false
declare VERBOSE=false
declare VERSION=""
declare OFFLINE_TARBALL=""

# ============================================================================
# LOGGING FUNCTIONS
# ============================================================================
log_debug() {
    if [[ "${VERBOSE}" == true ]]; then
        echo "[DEBUG] ${*}" >&2
    fi
}

log_info() {
    echo "[INFO] ${*}" >&2
}

log_warn() {
    echo "[WARN] ${*}" >&2
}

log_error() {
    echo "[ERROR] ${*}" >&2
}

die() {
    log_error "${*}"
    exit "${EXIT_FAILURE}"
}

# ============================================================================
# HELP DOCUMENTATION
# ============================================================================
help() {
    cat <<EOF
${SCRIPT_NAME} - Install ${BINARY_NAME}

USAGE:
    ${SCRIPT_NAME} [OPTIONS]

    Or via curl:
    curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash
    curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/install.sh | bash -s -- --version v1.0.0

DESCRIPTION:
    Downloads and installs the ${BINARY_NAME} binary from GitHub Releases.
    Auto-detects platform (linux/darwin/windows) and architecture (amd64/arm64).
    Verifies SHA256 checksums for integrity.

OPTIONS:
    -h, --help                  Show this help message and exit
    -v, --verbose               Enable verbose output
    -V, --version VERSION       Install a specific version (e.g., v1.0.0)
    --system                    Install to /usr/local/bin (requires sudo)
    --offline-tarball FILE      Install from a local tarball (skip download)

EXAMPLES:
    # Install latest version to ~/.local/bin
    ${SCRIPT_NAME}

    # Install specific version
    ${SCRIPT_NAME} --version v1.0.0

    # System-wide install
    ${SCRIPT_NAME} --system

    # Offline install from pre-downloaded tarball
    ${SCRIPT_NAME} --offline-tarball pi-go_1.0.0_linux_amd64.tar.gz

EXIT CODES:
    0    Success
    1    General failure
    2    Usage error

EOF
}

# ============================================================================
# CLEANUP AND SIGNAL HANDLING
# ============================================================================
cleanup() {
    local -ri exit_code=$?
    log_debug "Cleanup called with exit code: ${exit_code}"

    if [[ -n "${TEMP_DIR:-}" ]] && [[ -d "${TEMP_DIR}" ]]; then
        rm -rf "${TEMP_DIR}"
        log_debug "Removed temporary directory: ${TEMP_DIR}"
    fi

    exit "${exit_code}"
}

trap cleanup EXIT
trap 'die "Installation interrupted"' INT TERM

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================
check_dependencies() {
    local -a required_commands=("mktemp")
    local missing=false

    # In offline mode we only need tar/unzip; in online mode we need curl + sha256
    if [[ -z "${OFFLINE_TARBALL}" ]]; then
        required_commands+=("curl")
    fi

    for cmd in "${required_commands[@]}"; do
        if ! command -v "${cmd}" &> /dev/null; then
            log_error "Required command not found: ${cmd}"
            missing=true
        fi
    done

    if [[ "${missing}" == true ]]; then
        die "Missing required dependencies. Please install them and try again."
    fi
}

# Detect the sha256 command available on the system.
# Returns the command name via stdout.
detect_sha256_command() {
    if command -v sha256sum &> /dev/null; then
        echo "sha256sum"
    elif command -v shasum &> /dev/null; then
        echo "shasum"
    else
        echo ""
    fi
}

detect_os() {
    local os
    os="$(uname -s)"

    case "${os}" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        CYGWIN*|MINGW*|MSYS*) echo "windows" ;;
        *)       die "Unsupported operating system: ${os}" ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"

    case "${arch}" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              die "Unsupported architecture: ${arch}" ;;
    esac
}

archive_extension() {
    local -r os="${1}"

    if [[ "${os}" == "windows" ]]; then
        echo "zip"
    else
        echo "tar.gz"
    fi
}

# Fetch the latest release tag from the GitHub API.
fetch_latest_version() {
    local response
    response=$(curl -fsSL "${GITHUB_API}/releases/latest" 2>&1) \
        || die "Failed to fetch latest release from GitHub. Check your internet connection."

    local tag
    # Parse tag_name without requiring jq; works with grep + sed
    tag=$(echo "${response}" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/')

    if [[ -z "${tag}" ]]; then
        die "Could not determine latest version from GitHub API response."
    fi

    echo "${tag}"
}

# Download a file from a URL to a local path.
download_file() {
    local -r url="${1}"
    local -r dest="${2}"

    log_debug "Downloading: ${url}"
    if ! curl -fsSL -o "${dest}" "${url}"; then
        die "Failed to download: ${url}"
    fi
}

# Verify SHA256 checksum of a file against the checksums file.
verify_checksum() {
    local -r checksums_file="${1}"
    local -r target_file="${2}"
    local -r target_basename="${3}"

    local sha256_cmd
    sha256_cmd=$(detect_sha256_command)

    if [[ -z "${sha256_cmd}" ]]; then
        log_warn "No SHA256 tool found (sha256sum or shasum); skipping checksum verification."
        return 0
    fi

    # Extract the expected checksum from the checksums file
    local expected
    expected=$(grep "${target_basename}" "${checksums_file}" | awk '{print $1}')

    if [[ -z "${expected}" ]]; then
        die "Checksum entry not found for ${target_basename} in SHA256SUMS."
    fi

    # Compute the actual checksum
    local actual
    if [[ "${sha256_cmd}" == "sha256sum" ]]; then
        actual=$(sha256sum "${target_file}" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "${target_file}" | awk '{print $1}')
    fi

    if [[ "${expected}" != "${actual}" ]]; then
        die "Checksum mismatch for ${target_basename}. Expected: ${expected}, Got: ${actual}"
    fi

    log_info "Checksum verified: ${target_basename}"
}

# Extract binary from archive into destination directory.
extract_archive() {
    local -r archive="${1}"
    local -r dest_dir="${2}"
    local -r ext="${3}"

    log_debug "Extracting: ${archive} -> ${dest_dir}"

    case "${ext}" in
        tar.gz)
            tar -xzf "${archive}" -C "${dest_dir}"
            ;;
        zip)
            unzip -q -o "${archive}" -d "${dest_dir}"
            ;;
        *)
            die "Unsupported archive format: ${ext}"
            ;;
    esac
}

# Run a command with sudo if needed (skips sudo when already root).
maybe_sudo() {
    if [[ "$(id -u)" -eq 0 ]]; then
        "${@}"
    else
        sudo "${@}"
    fi
}

# Install the binary to the target directory.
install_binary() {
    local -r source="${1}"
    local -r dest_dir="${2}"

    # Ensure destination directory exists
    if [[ "${INSTALL_SYSTEM}" == true ]]; then
        if [[ ! -d "${dest_dir}" ]]; then
            maybe_sudo mkdir -p "${dest_dir}"
        fi
        maybe_sudo cp "${source}" "${dest_dir}/${BINARY_NAME}"
        maybe_sudo chmod 755 "${dest_dir}/${BINARY_NAME}"
    else
        mkdir -p "${dest_dir}"
        cp "${source}" "${dest_dir}/${BINARY_NAME}"
        chmod 755 "${dest_dir}/${BINARY_NAME}"
    fi

    log_info "Installed ${BINARY_NAME} to ${dest_dir}/${BINARY_NAME}"
}

# Check whether the install directory is on PATH and advise if not.
check_path() {
    local -r dir="${1}"

    if [[ ":${PATH}:" != *":${dir}:"* ]]; then
        log_warn "${dir} is not in your PATH."
        echo "" >&2
        echo "  Add it to your shell profile:" >&2
        echo "    export PATH=\"${dir}:\${PATH}\"" >&2
        echo "" >&2
    fi
}

# ============================================================================
# ARGUMENT PARSING
# ============================================================================
parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case "${1}" in
            -h|--help)
                help
                exit "${EXIT_SUCCESS}"
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -V|--version)
                [[ -z "${2:-}" ]] && die "Option ${1} requires an argument"
                VERSION="${2}"
                shift 2
                ;;
            --system)
                INSTALL_SYSTEM=true
                INSTALL_DIR="/usr/local/bin"
                shift
                ;;
            --offline-tarball)
                [[ -z "${2:-}" ]] && die "Option ${1} requires an argument"
                OFFLINE_TARBALL="${2}"
                shift 2
                ;;
            -*)
                die "Unknown option: ${1}. Use --help for usage information."
                ;;
            *)
                die "Unexpected argument: ${1}. Use --help for usage information."
                ;;
        esac
    done
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================
main() {
    parse_arguments "${@}"
    check_dependencies

    local os
    local arch
    os=$(detect_os)
    arch=$(detect_arch)

    log_info "Detected platform: ${os}/${arch}"

    local ext
    ext=$(archive_extension "${os}")

    # Set up temporary working directory
    TEMP_DIR=$(mktemp -d)
    log_debug "Temporary directory: ${TEMP_DIR}"

    if [[ -n "${OFFLINE_TARBALL}" ]]; then
        # ----------------------------------------------------------------
        # Offline installation from local tarball
        # ----------------------------------------------------------------
        [[ -f "${OFFLINE_TARBALL}" ]] || die "Tarball not found: ${OFFLINE_TARBALL}"
        [[ -r "${OFFLINE_TARBALL}" ]] || die "Tarball not readable: ${OFFLINE_TARBALL}"

        log_info "Installing from local tarball: ${OFFLINE_TARBALL}"

        # Determine extension from the filename
        local offline_ext
        if [[ "${OFFLINE_TARBALL}" == *.tar.gz ]]; then
            offline_ext="tar.gz"
        elif [[ "${OFFLINE_TARBALL}" == *.zip ]]; then
            offline_ext="zip"
        else
            die "Unsupported archive format. Expected .tar.gz or .zip"
        fi

        extract_archive "${OFFLINE_TARBALL}" "${TEMP_DIR}" "${offline_ext}"
    else
        # ----------------------------------------------------------------
        # Online installation from GitHub Releases
        # ----------------------------------------------------------------
        if [[ -z "${VERSION}" ]]; then
            log_info "Fetching latest version..."
            VERSION=$(fetch_latest_version)
        fi

        log_info "Installing ${BINARY_NAME} ${VERSION}"

        # Strip leading 'v' for the archive filename (goreleaser convention)
        local version_number
        version_number="${VERSION#v}"

        local archive_name
        archive_name="${BINARY_NAME}_${version_number}_${os}_${arch}.${ext}"

        local download_url
        download_url="${GITHUB_RELEASES}/download/${VERSION}/${archive_name}"

        local checksums_url
        checksums_url="${GITHUB_RELEASES}/download/${VERSION}/SHA256SUMS"

        # Download archive and checksums
        log_info "Downloading ${archive_name}..."
        download_file "${download_url}" "${TEMP_DIR}/${archive_name}"

        log_info "Downloading SHA256SUMS..."
        download_file "${checksums_url}" "${TEMP_DIR}/SHA256SUMS"

        # Verify checksum
        verify_checksum "${TEMP_DIR}/SHA256SUMS" "${TEMP_DIR}/${archive_name}" "${archive_name}"

        # Extract
        extract_archive "${TEMP_DIR}/${archive_name}" "${TEMP_DIR}" "${ext}"
    fi

    # Locate the binary in the extracted contents
    local binary_path
    binary_path="${TEMP_DIR}/${BINARY_NAME}"

    if [[ "${os}" == "windows" ]]; then
        binary_path="${TEMP_DIR}/${BINARY_NAME}.exe"
    fi

    if [[ ! -f "${binary_path}" ]]; then
        die "Binary not found after extraction. Expected: ${binary_path}"
    fi

    # Install
    install_binary "${binary_path}" "${INSTALL_DIR}"

    # Verify installation
    if "${INSTALL_DIR}/${BINARY_NAME}" --version &> /dev/null; then
        local installed_version
        installed_version=$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>&1 || true)
        log_info "Successfully installed: ${installed_version}"
    else
        log_info "Installation complete. Binary placed at: ${INSTALL_DIR}/${BINARY_NAME}"
    fi

    check_path "${INSTALL_DIR}"

    log_info "Done."
}

# ============================================================================
# SCRIPT ENTRY POINT
# ============================================================================
# Support both direct execution and piped execution (curl | bash)
main "${@:-}"
