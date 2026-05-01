# WIP: Config Refactoring

## Status
This is a work-in-progress refactoring to reorganize the configuration structure.

## Goal
Move from flat structure to nested structure under `diff:` key:

### Old Structure:
```yaml
ignoreDifferences: [...]
normalization: {...}
output: {...}
```

### New Structure:
```yaml
diff:
  ignoreDifferences: [...]
  normalization: {...}
  cli:
    display: side-by-side
    colorize: true
    ...
```

## What's Done
- ✅ Updated config types (Config, DiffConfig, CLIConfig)
- ✅ Updated normalizer to use new structure
- ✅ Updated validator to use new structure  
- ✅ Updated diff command to use new structure
- ✅ Updated example config files

## What's Needed
- ❌ Update all test files to use new config structure:
  - pkg/config/loader_test.go
  - pkg/config/validator_test.go
  - pkg/normalizer/normalizer_test.go
  - pkg/reporter/reporter_test.go (if needed)

## How to Fix Tests
Replace references like:
- `cfg.IgnoreDifferences` → `cfg.Diff.IgnoreDifferences`
- `cfg.Normalization` → `cfg.Diff.Normalization`
- `cfg.Output` → `cfg.Diff.CLI`

Update struct literals to wrap fields in `Diff: DiffConfig{...}`
