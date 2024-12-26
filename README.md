# Enrich Pocketsmith transactions for Bangkok Bank

## Why? 

By default, Bangkok Bank only has generic transaction descriptions such as "Payment for goods" since the web version only returns those and nothing else. 

The app, however, has rich transaction descriptions and by default saves those as receipts. 

This script is for parsing meta information from those receipts and enriching the transactions on Pocketsmith with it.

## How to use 

Currently it's very specific to my use case. 

The script expects a file with a bunch of meta information extracted from those receipts in the following format:

```
filename=IMG_0114;to=ARTIS COFFEE (THAILAND) LTD.;from=MR DAVID;amountTHB=130.00;amountOther=0;date=2024-12-26;time=12:15;bankref=123;txref=123
```

I am generating those from the receipts with Claude on my iOS device. 

Then to run the script: 

```
go run main.go -pocketsmith-token xxx -pocketsmith-transaction-account 123 -transaction-meta-file transactions.txt
```

```
[1/34] Processing: xxxx
Enriching transaction: 1160545915: Payment for Goods /Services -> ARTIS COFFEE (THAILAND) LTD.
```