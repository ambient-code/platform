#!/bin/bash

# Ambient Platform Python SDK Test Script
# This script sets up the environment and runs the Python SDK example

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_header() {
    echo -e "${BLUE}🐍 Ambient Platform Python SDK Test${NC}"
    echo -e "${BLUE}====================================${NC}"
}

# Check required environment variables
check_environment() {
    local missing_vars=()
    
    if [[ -z "${AMBIENT_TOKEN:-}" ]]; then
        missing_vars+=("AMBIENT_TOKEN")
    fi
    
    if [[ -z "${AMBIENT_PROJECT:-}" ]]; then
        missing_vars+=("AMBIENT_PROJECT")
    fi
    
    if [[ -z "${AMBIENT_API_URL:-}" ]]; then
        missing_vars+=("AMBIENT_API_URL")
    fi
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        print_error "Missing required environment variables:"
        echo
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        echo
        print_info "Please set all required environment variables:"
        echo
        echo "  export AMBIENT_TOKEN=\"your-bearer-token\""
        echo "  export AMBIENT_PROJECT=\"your-project-name\""  
        echo "  export AMBIENT_API_URL=\"https://your-api-endpoint.com\""
        echo
        print_info "Examples:"
        echo
        echo "  # Using OpenShift token (recommended):"
        echo "  export AMBIENT_TOKEN=\"\$(oc whoami -t)\""
        echo "  export AMBIENT_PROJECT=\"anynamespace\""
        echo "  export AMBIENT_API_URL=\"https://public-api-route-yournamespace.apps.rosa.xezue-pjejw-oy9.ag90.p3.openshiftapps.com\""
        echo
        echo "  # Using manual token:"
        echo "  export AMBIENT_TOKEN=\"sha256~_3FClshuberfakepO_BGI_tZg_not_real_token_Jv72pRN-r5o\""
        echo "  export AMBIENT_PROJECT=\"anynamespace\""
        echo "  export AMBIENT_API_URL=\"https://public-api-route-yournamespace.apps.rosa.xezue-pjejw-oy9.ag90.p3.openshiftapps.com\""
        echo
        print_warning "Then run this script again: ./test.sh"
        exit 1
    fi
    
    print_success "All required environment variables are set"
}

# Validate environment variables
validate_environment() {
    print_info "Validating environment variables..."
    
    # Check token format (should not contain AMBIENT_TOKEN= prefix)
    if [[ "${AMBIENT_TOKEN}" == *"AMBIENT_TOKEN="* ]]; then
        print_error "Invalid token format detected"
        echo
        print_info "Your token contains 'AMBIENT_TOKEN=' which will cause API errors."
        echo "Current token: ${AMBIENT_TOKEN}"
        echo
        print_info "Please fix your token by removing the duplicate prefix:"
        echo "export AMBIENT_TOKEN=\"${AMBIENT_TOKEN#*AMBIENT_TOKEN=}\""
        exit 1
    fi
    
    # Check if URL is valid format
    if [[ ! "${AMBIENT_API_URL}" =~ ^https?:// ]]; then
        print_warning "API URL should start with http:// or https://"
        print_info "Current URL: ${AMBIENT_API_URL}"
    fi
    
    # Check project name format (basic validation)
    if [[ ! "${AMBIENT_PROJECT}" =~ ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$ ]]; then
        print_warning "Project name should follow Kubernetes naming conventions (lowercase alphanumeric with hyphens)"
        print_info "Current project: ${AMBIENT_PROJECT}"
    fi
    
    print_success "Environment variables validated"
    
    # Display configuration
    echo
    print_info "Configuration:"
    echo "  API URL: ${AMBIENT_API_URL}"
    echo "  Project: ${AMBIENT_PROJECT}"
    echo "  Token length: ${#AMBIENT_TOKEN} characters"
    echo "  Token prefix: ${AMBIENT_TOKEN:0:12}..."
}

# Check if we're in the right directory
check_directory() {
    if [[ ! -f "pyproject.toml" ]] || [[ ! -d "ambient_platform" ]] || [[ ! -f "examples/main.py" ]]; then
        print_error "This script must be run from the python-sdk directory"
        echo
        print_info "Expected directory structure:"
        echo "  python-sdk/"
        echo "  ├── pyproject.toml"
        echo "  ├── ambient_platform/"
        echo "  └── examples/main.py"
        echo
        print_info "Please navigate to the correct directory:"
        echo "  cd /path/to/platform/components/ambient-sdk/python-sdk"
        echo "  ./test.sh"
        exit 1
    fi
    
    print_success "Running from correct directory: $(pwd)"
}

# Setup Python virtual environment
setup_venv() {
    print_info "Setting up Python virtual environment..."
    
    if [[ ! -d "venv" ]]; then
        print_info "Creating virtual environment..."
        python -m venv venv
        print_success "Virtual environment created"
    else
        print_success "Virtual environment already exists"
    fi
}

# Install dependencies
install_dependencies() {
    print_info "Installing dependencies..."
    
    # Activate virtual environment
    source venv/bin/activate
    
    # Install SDK in development mode
    pip install -e . > /dev/null 2>&1
    
    print_success "Dependencies installed successfully"
}

# Test SDK import
test_import() {
    print_info "Testing SDK import..."
    
    # Activate virtual environment
    source venv/bin/activate
    
    # Test basic import
    python -c "import ambient_platform; print('Import successful')" > /dev/null
    
    # Test specific imports
    python -c "
from ambient_platform import (
    AmbientClient, 
    CreateSessionRequest, 
    RepoHTTP,
    StatusPending,
    StatusCompleted
)
print('All imports successful')
" > /dev/null
    
    print_success "SDK imports working correctly"
}

# Run the example
run_example() {
    print_info "Running Python SDK example..."
    echo
    
    # Activate virtual environment and run example
    source venv/bin/activate
    python examples/main.py
}

# Main execution
main() {
    print_header
    echo
    
    # Run all checks and setup
    check_directory
    check_environment
    validate_environment
    echo
    
    setup_venv
    install_dependencies
    test_import
    echo
    
    print_success "Setup complete! Running example..."
    echo
    
    run_example
    
    echo
    print_success "Python SDK test completed successfully!"
}

# Run main function
main "$@"