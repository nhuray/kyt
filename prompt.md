I think it gonna be complicated to apply a global masking strategy for all use cases, so instead I think we should define a default masking rule (e.g `first N characters, last Y characters`) and being able to define rules based on regexp patterns.

The default rule will apply when no other rule pattern are matching.

The configuration will looks like that:

```yaml
# .kyt.yaml
diff:
  
  # Secret masking configuration (NEW in v0.4.0)
  # Masks Kubernetes Secret data/stringData values to prevent credential leaks in CI/CD logs
  secretMasking:
    # Enable/disable Secret masking (default: true)
    # When enabled, Secret values are masked in all diff outputs (unified, summary, markdown, TUI)
    enabled: true
    
    # Masking strategies
    strategies:
      # Masking strategy
      maskAll:
        description: Mask All characters
        example: "********** for 1234567890"
        # Character used for masking (default: "*")
        maskChar: "*"
      
      # Mask every character beside the X first and Y last characters
      maskMid:
        description: Mask every character beside the X first and Y last characters
        example: "12******90 for 1234567890"
        maskChar: "*"
        keepFirst: 2  # keep the 2 first characters
        keepLast: 2   # keep the 2 last characters

      # More complex masking strategy
      complexMask:
        description: Mask a sequence of X characters then keep Y character, then repeat
        example: "12*****8*****d**gh for 1234567890abcdefgh"
        maskChar: "*"
        keepFirst: 2  # keep the 2 first characters
        keepLast: 2   # keep the 2 last characters
        maskSequence: 5   # mask 5 characters then use  
        keepMid: 1    # keep 1 character between masking sequence
   
    # Masking rules: keep-first-last strategy
    rules:
        # Default rules applied when other rules can't be applied
        default:
          description: Default masking rule
          pattern: "^.{2}(?<maskAll>.+).{2}$"   # mask all characters beside the 2 first and 2 last characters
            
        # MongoDB Url masking rule
        mongodbUrl:
          description: Masking rule for MongoDB URL
          pattern: "^mongodb\+srv:\/\/.+:(?<maskMid>[^?]+)\?.+$"
       
        # Strike Keys masking rules
        stripe:
           description: Masking Rule for Strike API Keys
           pattern: "^[p|s]k_(live|test)_.{2}(?<complexMask>.+).{3}$"
```    


--- 

I like your design proposal, defining a mapping between captured-groups and masking strategy, it simplifies things and we don't need to define named strategies finally.

What we can do instead is to define the following build-in strategies (inspired by go-masker types):
 - `mask-F-E`: Keep first F, last E, mask middle. For instance: `sk-abc123xyz` (mask-4-4)	-> `sk-a***3xyz` 
 - `mask-F-E-S-M`: Keep first F, last E, then (mask S then keep M and repeat). For instance: `1234567890abcdefgh` (mask-2-2-5-1) -> `12*****8*****d**gh`

Here is a refined version of you excellent proposal:

```yaml
diff:
  secretMasking:
    enabled: true
        
    # Masking rules (evaluated in order, first match wins)
    rules:
      # MongoDB connection strings
      - name: mongodb-url
        description: Mask password in MongoDB connection strings
        pattern: '^mongodb(\+srv)?://[^:]+:(?<password>[^@]+)@.+$'
        groupMasks:
          password: mask-0-0  # mask everything (0 first and last characters kept)
        # Rest of URL kept as-is
      
      # Stripe API keys
      - name: stripe-keys
        description: Mask Stripe API keys
        pattern: '^(pk|sk)_(live|test)_(?<key>.+)$'
        groupMasks:
          key: mask-2-3   # keep 2 first characters and 3 last characters 
      
      # PostgreSQL connection strings
      - name: postgres-url
        description: Mask password in PostgreSQL URLs
        pattern: '^postgres(ql)?://[^:]+:(?<password>[^@]+)@(?host[^?]+)?$'
        groupMasks:
          password: mask-1-1  #  keep 1 first character and 1 last character 
          host: mask-1-1-3-1  #  keep 1 first character and 1 last character, mask 3 character, then show 1 and repeat
      
      # Generic secrets (default fallback)
      - name: default
        description: Default masking for all other secrets
        pattern: '^(?<value>.+)$'  # match everything
        groupMasks:
          value: mask-2-2   # keep 2 first characters and 3 last characters 
```

We should fail fast before performing the diff if the strategy does not match `mask-\d+-\d+(-\d+-\d+)?`.

What do you think ?

----
Answer to you questions:

1. Mask Character - Global or Per-Rule?: Option C (both)
2. Unmatched Capture Groups: Option A (fail-fast)
3. Non-Captured Parts of the Match: Correct, we only mask what is captured by a group !
4. Strategy Validation Regex: Yes we have to validate the semantic
5. Edge Case: Pattern Matches but No Captures: Option C (fail-fast)
6. Pattern Match Failure: Option B (mask everything) with a warning
7. Regex Pattern Validation: Option A (fail-fast)
8. Sequenced Masking Behavior Clarification: Yes, we repeat the sequence so the remaining `ef` must be replace by `**` 
9. Performance & Caching: Yes
10. Configuration Example Location: Update the existing documentation and example but don't create new ones.

Does `groupMasks` make sense? Do you have a better terminology ? 
