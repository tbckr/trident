module github.com/tbckr/trident

go 1.26.0

require (
	codeberg.org/miekg/dns v0.6.97 // Revisit once nixpkgs-unstable has go_1_27 >= 1.27.4 (GO-2026-6859): v0.6.98+ needs Go 1.27, so raise Go first (go.mod, .go-version, flake.nix); never v0.6.101, a side-branch tag
	github.com/imroc/req/v3 v3.61.0
	github.com/jarcoal/httpmock v1.4.2
	github.com/olekukonko/tablewriter v1.1.5
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.1.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.12.1
	golang.org/x/net v0.63.0
	golang.org/x/term v0.48.0
	golang.org/x/time v0.16.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/andybalholm/brotli v1.2.7 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/displaywidth v0.10.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.6.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/icholy/digest v1.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.20.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/olekukonko/cat v0.0.0-20250911104152-50322a0618f6 // indirect
	github.com/olekukonko/errors v1.2.0 // indirect
	github.com/olekukonko/ll v0.1.6 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.61.0 // indirect
	github.com/refraction-networking/utls v1.8.2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
