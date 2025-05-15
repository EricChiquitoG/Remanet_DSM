# Use an official Python base image
FROM python:3.13-slim

# Set a working directory
WORKDIR /app

# Copy your requirements and install them
COPY requirements.txt .
RUN pip install -r requirements.txt

# Copy your script into the container
COPY opt.py .
COPY optClient.py .
COPY optimizationData.json .
COPY DSM_pb2_grpc.py .
COPY DSM_pb2.py .

# Expose the port the server listens on (optional, but good practice)
EXPOSE 50060

# Default command to run your script
CMD ["python", "-u", "optClient.py"]