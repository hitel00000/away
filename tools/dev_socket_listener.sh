#!/usr/bin/env bash
set -e

SOCK=/run/irc-companion.sock
rm -f $SOCK

socat - UNIX-LISTEN:$SOCK,fork