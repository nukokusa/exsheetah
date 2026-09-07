# exsheetah

exsheetah is a tool for exporting data from local xlsx files to YAML or JSON files.

It is a sibling of [sheetah](https://github.com/nukokusa/sheetah): where sheetah
reads sheets from Google Sheets, exsheetah reads sheets from an xlsx file on
disk. The configuration format is otherwise the same.

## Usage

```
Usage: exsheetah <command> [flags]

Flags:
  -h, --help                       Show context-sensitive help.
  -c, --config="exsheetah.yaml"    Load configuration from FILE
  -v, --version                    Show version.

Commands:
  validate [flags]
    Validate configuration file.

  export --file=STRING [flags]
    Export sheets to files.
```

```
Usage: exsheetah export --file=STRING [flags]

Export sheets to files.

Flags:
      --file=STRING                Path to the xlsx file to export from
                                   ($EXSHEETAH_FILE)
      --format="yaml"              Export format [yaml,json]
      --dir="."                    Export directory
      --sheets=SHEETS,...          Sheet names (matching a SheetConfig name) to
                                   export; exports every configured sheet when
                                   omitted
```

Each configured sheet is written to `<dir>/<name>.yaml` or `<dir>/<name>.json`,
depending on `--format`.

`--sheets` restricts which configured sheets are exported, by their `name` in
the configuration file (e.g. `--sheets=weapon,item`). When omitted, every
sheet in the configuration is exported. Naming a sheet that isn't in the
configuration is an error.

## Configuration

The configuration file is YAML format. Describe the structure of the table.

```yaml
timezone: Asia/Tokyo
sheets:
  - name: weapon
    range: A1:D10
    columns:
      - name: id
        type: number
      - name: name
        type: string
      - name: damage
        type: number
      - name: release_date
        type: timestamp
  - name: item
    columns:
      - name: id
        type: number
      - name: name
        type: string
      - name: consumable
        type: boolean
```

- `timezone` is the IANA timezone name (e.g. `Asia/Tokyo`) used to interpret
  date/time cell values. Optional; defaults to `UTC` when omitted.
- `name` is both the entry's name (and the output file's base name) and,
  unless `sheet` is set, the name of the worksheet to read within the xlsx
  file.
- `sheet` optionally overrides the worksheet name, when it should differ
  from `name`.
- `range` is an A1-style cell range (e.g. `A1:D10`) restricting which cells
  are read. When omitted, the sheet's whole used range is read.
- `columns` describes the header row: the first row within the read range
  that contains one of these column names is treated as the header, rows
  before it are skipped, and rows after it become data rows.

The `type` specifies the data type of the column. The following types can be
used:
- number
- string
- boolean
- timestamp

If the sheet value does not match the type, it will not be output.

### Timestamps

xlsx stores date/time values as plain serial numbers with no timezone of
their own (unlike Google Sheets, which has a per-spreadsheet timezone
setting). exsheetah interprets them using the configuration file's
`timezone` field (default `UTC`), and outputs them as RFC 3339 strings.

A `timestamp` column also accepts a plain text cell containing a date/time
string, parsed the same way as sheetah.

## Author

Copyright (c) 2025 Daisuke Nagashima

## LICENSE

MIT
