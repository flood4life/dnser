# DNS Maintainer

> DNS done GitOps way

## How it works

```yaml
apiVersion: 1
config:
- ip: 127.0.0.1
  domain: example.org
  aliases:
  - foo.example.org:
    - bar.example.org
    - baz.example.org
  - foobar.example.org    
```

The configuration consists of a number of items. 
Each item must contain `ip`, `domain` and `aliases`. `aliases` is a list of trees.

DNSer will create an A record of `domain` to `ip`. Then it will create ALIAS tree roots to `domain`,
and then alias each tree node to their parent.

DNSer will also delete all records that resolve to `domain` but not present in any of the `aliases` trees.
