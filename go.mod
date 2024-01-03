module github.com/goclarum/clarum/http

go 1.21

// until core is published
require github.com/goclarum/clarum/core v0.0.0
require github.com/goclarum/clarum/json v0.0.0

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/goclarum/clarum/core => ../clarum-core
replace github.com/goclarum/clarum/json => ../clarum-json
