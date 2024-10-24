# Configuration Options

## Global Configuration Options

| Option                  | Environment Variable              | Key in config file      | Command Line Flag           | Description                                                             | Possible Values                           | Default Value |
|-------------------------|-----------------------------------|-------------------------|-----------------------------|-------------------------------------------------------------------------|-------------------------------------------|---------------|
| Verbose Output          |                                   |                         | `--verbose`, `-v`           | Whether to output verbose information.                                  | `true`, `false`                           | `false`       |
| PAP Level               | `TRIDENT_PAP_LEVEL`               | `papLevel`              | `--pap-level`               | The environment level of the Permissible Actions Protocol (PAP) to use. | `RED`, `AMBER`, `GREEN`, `CLEAR`, `WHITE` | `WHITE`       |
| Disable Domain Brackets | `TRIDENT_DISABLE_DOMAIN_BRACKETS` | `disableDomainBrackets` | `--disable-domain-brackets` | Whether to disable the use of domain brackets.                          | `true`, `false`                           | `false`       |
