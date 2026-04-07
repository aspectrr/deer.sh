#!/bin/bash
set -e

if systemctl is-active --quiet deer-daemon 2>/dev/null; then
    systemctl stop deer-daemon
fi
if systemctl is-enabled --quiet deer-daemon 2>/dev/null; then
    systemctl disable deer-daemon
fi
