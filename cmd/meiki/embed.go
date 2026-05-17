package main

import "github.com/arikbautista/meiki/instructions"

var meikiMD = instructions.MeikiMD

var stopHookSnippet = `{
  "hooks": {
    "Stop": [{
      "matcher": "*",
      "hooks": [{"type": "command", "command": "meiki review --silent"}]
    }]
  }
}`
