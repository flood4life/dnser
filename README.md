# dnser

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

`dnser` will create an A record of `domain` to `ip`. Then it will create ALIAS tree roots to `domain`,
and then alias each tree node to their parent.

`dnser` will also delete all records that resolve to `domain` but not present in any of the `aliases` trees.

## Usage

### Go package

```go
config := config.LoadFromString(yamlString)
r53Adapter := adapter.NewRoute53(id, secret)
records, err := r53Adapter.List(context.Background())
if err != nil {
    panic(err)
}
m := massager.Massager{
    Desired: config,
    Current: records,
}
chset := m.CalculateNeededActions()
err = r53adapter.Process(context.Background(), chset)
if err != nil {
    panic(err)
}
```
