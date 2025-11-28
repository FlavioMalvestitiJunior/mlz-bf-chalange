#!/bin/bash
# Cron script to run S3 importer every 10 minutes
# Add to crontab: */10 * * * * /path/to/run-s3-importer.sh

cd "$(dirname "$0")"
docker-compose run --rm s3-importer
