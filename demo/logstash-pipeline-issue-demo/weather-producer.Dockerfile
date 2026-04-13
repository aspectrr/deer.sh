FROM python:3.11-slim
RUN pip install --no-cache-dir kafka-python==2.0.2
COPY weather-producer.py /weather-producer.py
CMD ["python3", "/weather-producer.py"]
