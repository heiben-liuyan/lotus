#!/bin/sh

# only can call on deploy node
lotus net connect $(lotus --repo=/data/lotus/dev/.ldt0111 net listen)
