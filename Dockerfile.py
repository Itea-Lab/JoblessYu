FROM python:3.11-slim

# Enviroment variables
# - PYTHONDONTWRITEBYTECODE: Prevent Python from generating .pyc files
# - PYTHONUNBUFFERED: Ensure that the output log is written directly to stdout/stderr and not buffered
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

# Set the working directory in the container
WORKDIR /app

# Copy requirements.txt
COPY requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code to /app
COPY . .

# Run script
CMD ["python", "JoblessYu.py"]
