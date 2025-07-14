
```drone
seisu a = 10;
seisu b = xF; // 15
ronri c = shinri;
c;
```
```bash
go run main.go demo_vars.drone
```
```
true
```

```drone
seisu a = (10 + 2) * 5 / 2 - 1; // 29
seisu b = 10 % 3; // 1

a + b; // 30
```


```bash
go run main.go demo_arithmetic.drone
```

```
30
```

```drone
ronri a = (10 > 5) ^ (2 == 1);
ronri b = ~a; 

b;
```

```bash
go run main.go demo_logic.drone
```

```
true
```


```drone

seisu result = 0;
seisu check = 5;

sorenara (check > 10) {
    result = 1;
} sorenara (check == 5) {
    result = 2; 
} sorenara {
    result = 3;
}

result;
```

```bash
go run main.go demo_if.drone
```

```
2
```



```drone

seisu result = 0;
shuki i = 0:10 {
    sorenara (i == 1) {
        shushi; 
    }
    result = result + i;
    sorenara (i == 4) {
        kido; 
    }
}

result;
```

```bash
go run main.go demo_loop.drone
```

```
9
```

```drone

kansu factorial(n) {
    sorenara (n == 0) {
        modoru 1;
    }
    modoru n * factorial(n - 1);
}

factorial(5);
```

```bash
go run main.go demo_functions.drone
```

```
120
```

```drone

seisu fib = kansu(x) {
    sorenara (x < 2) {
        modoru x;
    }
    modoru fib(x - 1) + fib(x - 2);
};

fib(10);
```

```bash
go run main.go fibonacci.drone
```

```
55
```

```drone

hairetsu a = [10, 20, 30];
seisu b = a[1]; 

seisu sum = a[0] + a[1] + a[2]; // 10 + 20 + 30 = 60
sum;
```

```bash
go run main.go demo_arrays.drone
```

```
60
```



```drone

rippotai pos = {11, 22, 33, uso};

seisu x = pos=>x;
seisu y = pos=>y;
seisu z = pos=>z;

x + y + z;
```

```bash
go run main.go demo_rippotai.drone
```

```
66
```

```drone

ruikei { [1, 2] };
```

```bash
go run main.go demo_ruikei.drone
```

```
hairetsu
```
