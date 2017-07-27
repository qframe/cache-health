#!/bin/bash

curl -sH "Accept: application/json" localhost:8123/_health |jq .
