# ==============================================================================
# Stage 1: Build & Dependency Preparation Stage
# ==============================================================================
FROM python:3.12-slim-bookworm AS builder

# Prevent Python bytecode generation during build
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /build

# Install build dependencies for stripping and compiling if needed
RUN apt-get update && apt-get install -y --no-install-recommends \
    binutils \
    && rm -rf /var/lib/apt/lists/*

# Create virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy requirements and install
COPY requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --upgrade pip setuptools wheel && \
    pip install --no-compile -r requirements.txt

# Aggressively shrink virtual environment
RUN set -eux; \
    # 1. Keep only the active architecture binary in tls_client (saves ~25MB)
    ARCH=$(uname -m); \
    case "$ARCH" in \
        x86_64) TLS_ARCH="x86" ;; \
        aarch64|arm64) TLS_ARCH="arm64" ;; \
        *) TLS_ARCH="x86" ;; \
    esac; \
    find /opt/venv/lib/python*/site-packages/tls_client/dependencies/ -type f \
        ! -name "tls-client-${TLS_ARCH}.so" \
        ! -name "__init__.py" \
        -delete; \
    # 2. Delete C/C++ source, headers, and Cython intermediate files shipped in wheels (saves ~10MB) \
    find /opt/venv/lib/python*/site-packages/ -type f \( \
        -name "*.c" -o \
        -name "*.h" -o \
        -name "*.hpp" -o \
        -name "*.pxd" -o \
        -name "*.pyx" -o \
        -name "*.pyi" \
    \) -delete; \
    # 3. Strip all ELF binaries and shared object files (.so) \
    find /opt/venv/lib/python*/site-packages/ -type f -name "*.so*" -exec strip --strip-unneeded {} + 2>/dev/null || true; \
    # 4. Remove test suites, docs, build caches, and unnecessary tooling packages \
    find /opt/venv/lib/python*/site-packages/ -type d \( \
        -name "tests" -o \
        -name "test" -o \
        -name "testing" -o \
        -name "__pycache__" -o \
        -name "f2py" -o \
        -name "idlelib" -o \
        -name "tkinter" \
    \) -exec rm -rf {} + 2>/dev/null || true; \
    # 5. Remove dist-info extra metadata records \
    find /opt/venv/lib/python*/site-packages/ -type f \( \
        -name "RECORD" -o \
        -name "INSTALLER" -o \
        -name "REQUESTED" -o \
        -name "direct_url.json" \
    \) -delete; \
    # 6. Remove build tooling from runtime venv \
    rm -rf \
        /opt/venv/lib/python*/site-packages/pip* \
        /opt/venv/lib/python*/site-packages/setuptools* \
        /opt/venv/lib/python*/site-packages/wheel* \
        /opt/venv/bin/pip* \
        /opt/venv/bin/wheel*


# ==============================================================================
# Stage 2: Production Minimal Runtime Stage
# ==============================================================================
FROM python:3.12-slim-bookworm AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH"

# Install only essential runtime certificates and prune OS layers aggressively
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && apt-get clean \
    && rm -rf \
        /var/lib/apt/lists/* \
        /var/cache/apt/* \
        /var/cache/debconf/* \
        /var/log/* \
        /tmp/* \
        /var/tmp/* \
        /usr/share/doc/* \
        /usr/share/man/* \
        /usr/share/info/* \
        /usr/share/locale/* \
    # Strip unused standard library components from base python \
    && rm -rf \
        /usr/local/lib/python3.12/idlelib \
        /usr/local/lib/python3.12/tkinter \
        /usr/local/lib/python3.12/turtledemo \
        /usr/local/lib/python3.12/ensurepip \
        /usr/local/lib/python3.12/pydoc_data \
        /usr/local/lib/python3.12/unittest \
        /usr/local/lib/python3.12/test \
        /usr/local/lib/python3.12/lib2to3 \
        /usr/local/lib/python3.12/config-3.12-x86_64-linux-gnu \
        /usr/local/lib/python3.12/site-packages/pip* \
        /usr/local/lib/python3.12/site-packages/setuptools* \
        /usr/local/lib/python3.12/site-packages/wheel* \
        /usr/local/bin/pip* \
        /usr/local/bin/idle* \
        /usr/local/bin/pydoc* \
        /usr/local/bin/2to3*

# Create non-root group and user
RUN groupadd --system --gid 10001 appgroup && \
    useradd --system --uid 10001 --gid appgroup --create-home --home-dir /app appuser

WORKDIR /app

# Copy optimized virtual environment from builder
COPY --from=builder --chown=appuser:appgroup /opt/venv /opt/venv

# Copy application source code
COPY --chown=appuser:appgroup JoblessYu.py .

# Switch to non-root execution
USER 10001:10001

# Run the Python scraper
CMD ["python", "JoblessYu.py"]

