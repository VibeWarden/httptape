#!/bin/bash
# Toggle the upstream service on/off to demo cache fallback
if docker compose ps upstream | grep -q "running"; then
    echo "Stopping upstream..."
    docker compose stop upstream
else
    echo "Starting upstream..."
    docker compose start upstream
fi
