FROM ollama/ollama:latest

# Install certificates and dos2unix to fix Windows line endings
RUN apt-get update && \
    apt-get install -y ca-certificates openssl dos2unix && \
    rm -rf /var/lib/apt/lists/*

# Add the registry certificate to the trust store
RUN echo -n | openssl s_client -showcerts -connect registry.ollama.ai:443 2>/dev/null | \
    sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > /usr/local/share/ca-certificates/ollama.crt && \
    update-ca-certificates

ENV OLLAMA_INSECURE=1

COPY ollama-entrypoint.sh /usr/bin/ollama-entrypoint.sh

# Convert the script to Unix line endings and ensure it is executable
RUN dos2unix /usr/bin/ollama-entrypoint.sh && \
    chmod +x /usr/bin/ollama-entrypoint.sh
    
ENTRYPOINT ["/usr/bin/ollama-entrypoint.sh"]
