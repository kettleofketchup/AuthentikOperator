# Certificate management recipes
# Call as: just certs::install

# Install certificates from docker/certificates/
install:
    @echo "Installing certificates from docker/certificates/..."
    @if ls docker/certificates/*.pem 1>/dev/null 2>&1; then \
        sudo cp docker/certificates/*.pem /usr/local/share/ca-certificates/; \
        sudo update-ca-certificates; \
        echo "Certificates installed"; \
    else \
        echo "No .pem files found in docker/certificates/"; \
    fi
