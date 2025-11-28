@echo off
REM Windows batch script to run S3 importer
REM Schedule with Task Scheduler to run every 10 minutes

cd /d %~dp0
docker-compose run --rm s3-importer
