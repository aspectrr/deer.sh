#!/bin/bash
set -e

if systemctl is-active --quiet fluid-daemon 2>/dev/null; then
    systemctl stop fluid-daemon
fi
if systemctl is-enabled --quiet fluid-daemon 2>/dev/null; then
    systemctl disable fluid-daemon
fi
