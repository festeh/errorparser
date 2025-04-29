```
error[E0308]: mismatched types
 --> src/main.rs:5:5
  |
5 |     let y: i32 = x;
  |     ^^^^^^^^^^^ expected `i32`, found `&str`
  |
  = note: expected type `i32`
             found type `&str`

error: aborting due to previous error

For more information about this error, try `rustc --explain E0308`.
```

```
warning: unused variable: `x`
 --> src/another.rs:2:9
  |
2 |     let x = 5;
  |         ^ help: if this is intentional, prefix it with an underscore: `_x`
  |
  = note: `#[warn(unused_variables)]` on by default

warning: `my_crate` (lib) generated 1 warning
```
