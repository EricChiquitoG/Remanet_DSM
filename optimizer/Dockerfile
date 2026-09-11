# Use an official Python base image
FROM python:3.12-alpine

# Set a working directory
WORKDIR /app

# Copy your requirements and install them
COPY requirements.txt .

RUN apk add --no-cache \
    build-base \
    musl-dev \
    python3-dev \
    libffi-dev \
    openssl-dev \
    cargo \
    g++

RUN pip install -r requirements.txt

# Copy your script into the container
COPY opt_DAG.py .
COPY optClient_DAG.py .
COPY optimizationData.json .
COPY DSM_pb2_grpc.py .
COPY DSM_pb2.py .

# Expose the port the server listens on (optional, but good practice)
EXPOSE 50060

# Default command to run your script
CMD ["python", "-u", "optClient_DAG.py"]