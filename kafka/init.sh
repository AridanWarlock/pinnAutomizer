#!/bin/bash
set -e

BROKER="kafka:9094"
TOPICS_FILE="/kafka/topics.yaml"

echo "=== Kafka Topic Initializer ==="

# Ждём готовности Kafka
echo "Waiting for Kafka to be ready..."
until /opt/kafka/bin/kafka-broker-api-versions.sh --bootstrap-server "$BROKER" &>/dev/null; do
    echo "  Kafka not ready yet, retrying in 3s..."
    sleep 3
done
echo "Kafka is ready!"

# Создаём топики
echo ""
echo "Reading topics from $TOPICS_FILE..."

# Читаем YAML и создаём топики
# Используем yq если есть, если нет — простой grep + awk
create_topics() {
    local current_topic=""
    local current_partitions=""

    while IFS= read -r line; do
        # Парсим имя топика
        if echo "$line" | grep -q "name:"; then
            current_topic=$(echo "$line" | sed -E 's/.*name:\s*["'\'']?([^"'\''#]+)["'\'']?.*/\1/' | xargs)
        fi

        # Парсим количество партиций
        if echo "$line" | grep -q "partitions:"; then
            current_partitions=$(echo "$line" | grep -o '[0-9]\+')
        fi

        # Когда собрали имя и партиции — создаём топик
        if [ -n "$current_topic" ] && [ -n "$current_partitions" ]; then
            echo ""
            echo "Creating topic: $current_topic (partitions: $current_partitions)"

            /opt/kafka/bin/kafka-topics.sh \
                --bootstrap-server "$BROKER" \
                --create \
                --if-not-exists \
                --topic "$current_topic" \
                --partitions "$current_partitions" \
                --replication-factor 1

            echo "  Done: $current_topic"

            # Сбрасываем для следующего топика
            current_topic=""
            current_partitions=""
        fi
    done < "$TOPICS_FILE"
}

create_topics

echo ""
echo "=== All topics created successfully ==="
echo ""
echo "Current topics:"
/opt/kafka/bin/kafka-topics.sh --bootstrap-server "$BROKER" --list