# Built-in Functions Reference

go-stencil provides a comprehensive set of built-in functions for data manipulation, formatting, and document control.

## Function Categories

- [Data Functions](#data-functions)
- [String Functions](#string-functions)
- [Number Functions](#number-functions)
- [Date Functions](#date-functions)
- [Formatting Functions](#formatting-functions)
- [Control Functions](#control-functions)
- [Document Functions](#document-functions)
- [Type Conversion Functions](#type-conversion-functions)

## Data Functions

### empty
Checks if a value is empty (nil, empty string, empty array, etc.)

**Syntax:** `empty(value)`

**Examples:**
```
{{if empty(notes)}}No notes provided{{end}}
{{if not(empty(items))}}Items found{{end}}
```

### coalesce
Returns the first non-empty value from the arguments

**Syntax:** `coalesce(value1, value2, ..., default)`

**Examples:**
```
{{coalesce(user.nickname, user.name, "Guest")}}
{{coalesce(product.salePrice, product.regularPrice, 0)}}
```

### list
Creates a list/array from the provided arguments

**Syntax:** `list(item1, item2, ...)`

**Examples:**
```
{{for color in list("red", "green", "blue")}}
  - {{color}}
{{end}}
```

### data
Returns the entire template data context

**Syntax:** `data()`

**Examples:**
```
{{data()}}  // Outputs all template data (useful for debugging)
```

### map
Extracts a specific field from each item in a collection

**Syntax:** `map(fieldName, collection)`

**Examples:**
```
{{map("price", products)}}  // Returns [19.99, 29.99, 39.99]
{{sum(map("quantity", orderItems))}}  // Sum all quantities
```

## String Functions

### str
Converts any value to a string

**Syntax:** `str(value)`

**Examples:**
```
{{str(123)}}  // "123"
{{str(true)}}  // "true"
```

### lowercase
Converts string to lowercase

**Syntax:** `lowercase(text)`

**Examples:**
```
{{lowercase("HELLO WORLD")}}  // "hello world"
{{lowercase(product.name)}}
```

### uppercase
Converts string to uppercase

**Syntax:** `uppercase(text)`

**Examples:**
```
{{uppercase("hello world")}}  // "HELLO WORLD"
{{uppercase(customer.country)}}
```

### titlecase
Converts string to title case (first letter of each word capitalized)

**Syntax:** `titlecase(text)`

**Examples:**
```
{{titlecase("john doe")}}  // "John Doe"
{{titlecase(book.title)}}
```

### join
Joins array elements with a separator

**Syntax:** `join(array, separator)`

**Examples:**
```
{{join(tags, ", ")}}  // "urgent, important, review"
{{join(list("A", "B", "C"), " | ")}}  // "A | B | C"
```

### joinAnd
Joins array elements with commas and "and" before the last item

**Syntax:** `joinAnd(array)`

**Examples:**
```
{{joinAnd(list("Tom", "Jane", "Bob"))}}  // "Tom, Jane and Bob"
{{joinAnd(features)}}  // "fast, reliable and secure"
```

### replace
Replaces occurrences of a substring

**Syntax:** `replace(text, old, new)`

**Examples:**
```
{{replace("Hello World", "World", "Universe")}}  // "Hello Universe"
{{replace(phone, "-", "")}}  // Remove dashes
```

### length
Returns the length of a string, array, or map

**Syntax:** `length(value)`

**Examples:**
```
{{length("Hello")}}  // 5
{{length(items)}}  // Number of items
{{if length(password) < 8}}Password too short{{end}}
```

## Number Functions

### integer
Converts value to an integer

**Syntax:** `integer(value)`

**Examples:**
```
{{integer("123")}}  // 123
{{integer(3.14)}}  // 3
```

### decimal
Converts value to a decimal number

**Syntax:** `decimal(value)`

**Examples:**
```
{{decimal("3.14")}}  // 3.14
{{decimal(price)}}
```

### round
Rounds a number to the nearest integer

**Syntax:** `round(number)`

**Examples:**
```
{{round(3.14)}}  // 3
{{round(3.67)}}  // 4
{{round(price)}}
```

### floor
Rounds a number down to the nearest integer

**Syntax:** `floor(number)`

**Examples:**
```
{{floor(3.99)}}  // 3
{{floor(total)}}
```

### ceil
Rounds a number up to the nearest integer

**Syntax:** `ceil(number)`

**Examples:**
```
{{ceil(3.01)}}  // 4
{{ceil(shippingWeight)}}
```

### sum
Calculates the sum of numbers

**Syntax:** `sum(numbers...)`

**Examples:**
```
{{sum(1, 2, 3)}}  // 6
{{sum(map("price", items))}}  // Sum of all prices
```

## Date Functions

### date
Formats a date/time value

**Syntax:** `date(format, dateValue)`

**Format patterns (Go time format):**
- `"2006"` - Year
- `"01"` - Month (01-12)
- `"02"` - Day
- `"15"` - Hour (24-hour)
- `"04"` - Minute
- `"05"` - Second
- `"Mon"` - Weekday abbreviation
- `"Monday"` - Full weekday
- `"Jan"` - Month abbreviation
- `"January"` - Full month

**Examples:**
```
{{date("01/02/2006", orderDate)}}  // "03/15/2024"
{{date("Monday, January 2, 2006", createdAt)}}  // "Friday, March 15, 2024"
{{date("15:04:05", timestamp)}}  // "14:30:00"
```

## Formatting Functions

### format
Formats values using printf-style formatting

**Syntax:** `format(pattern, value)`

**Common patterns:**
- `"%.2f"` - Two decimal places
- `"%d"` - Integer
- `"%s"` - String
- `"%05d"` - Zero-padded integer
- `"%.0f"` - No decimal places

**Examples:**
```
{{format("%.2f", price)}}  // "19.99"
{{format("Order #%05d", orderNumber)}}  // "Order #00042"
{{format("%s: %d items", category, count)}}  // "Electronics: 15 items"
```

### formatWithLocale
Formats values with a specific locale

**Syntax:** `formatWithLocale(locale, pattern, value)`

**Examples:**
```
{{formatWithLocale("de-DE", "%.2f", price)}}  // "19,99" (German format)
{{formatWithLocale("fr-FR", "%.2f", total)}}  // "1 234,56"
```

### currency
Formats a number as currency (locale-aware)

**Syntax:** `currency(amount)`

**Examples:**
```
{{currency(19.99)}}  // "$19.99" (or locale-specific format)
{{currency(total)}}
```

### percent
Formats a number as a percentage

**Syntax:** `percent(value)`

**Examples:**
```
{{percent(0.15)}}  // "15%"
{{percent(growthRate)}}  // "23.5%"
```

## Control Functions

### switch
Evaluates a value and returns the corresponding result

**Syntax:** `switch(value, case1, result1, case2, result2, ..., default)`

**Examples:**
```
{{switch(status, "active", "✓", "pending", "⏳", "inactive", "✗", "?")}}
{{switch(userType, "admin", "Full Access", "user", "Limited Access", "No Access")}}
```

### contains
Checks if a collection contains an item

**Syntax:** `contains(item, collection)`

**Examples:**
```
{{if contains("admin", user.roles)}}Admin menu{{end}}
{{if contains(country, list("US", "CA", "MX"))}}North America{{end}}
```

### range
Generates a sequence of numbers

**Syntax:** `range(start, end)`

**Examples:**
```
{{for num in range(1, 5)}}
  {{num}}. Item
{{end}}
// Outputs: 1. Item, 2. Item, 3. Item, 4. Item
```

## Document Functions

### pageBreak
Inserts a page break in the document

**Syntax:** `pageBreak()`

**Examples:**
```
{{pageBreak}}
// Content after this appears on a new page
```

### hideRow
Hides the current table row (used within table loops)

**Syntax:** `hideRow()`

**Examples:**
```
{{for item in items}}
{{if item.hidden}}
{{hideRow()}}
{{else}}
| {{item.name}} | {{item.price}} |
{{end}}
{{end}}
```

### hideColumn
Hides a table column

**Syntax:** `hideColumn()` or `hideColumn(strategy)`

**Strategies:**
- `"resize-last"` - Resize the last column (default)
- `"resize-first"` - Resize the first column
- `"resize-all"` - Distribute space among all columns

**Examples:**
```
{{if not(showPrices)}}
{{hideColumn}}  // Hides the price column
{{end}}

{{hideColumn("resize-first")}}
```

### html
Renders HTML content as formatted text

**Syntax:** `html(htmlContent)`

**Supported tags:**
- `<b>`, `<strong>` - Bold
- `<i>`, `<em>` - Italic
- `<u>` - Underline
- `<s>`, `<strike>` - Strikethrough
- `<sub>` - Subscript
- `<sup>` - Superscript
- `<br>` - Line break
- `<span style="">` - Custom styling
- `<a href="">` - Hyperlinks
- Lists: `<ul>`, `<ol>`, `<li>`

**Examples:**
```
{{html("<b>Important:</b> Please <u>review</u> carefully")}}
{{html("First line<br>Second line<br>Third line")}}  // Line breaks
{{html(product.description)}}  // If description contains HTML
```

### xml
Inserts raw XML content (advanced use)

**Syntax:** `xml(xmlContent)`

**Examples:**
```
{{xml("<w:br/>")}}  // Word line break
{{xml(customXmlElement)}}
```

### replaceLink
Replaces a hyperlink URL

**Syntax:** `replaceLink(url)`

**Examples:**
```
{{replaceLink("https://example.com/product/" + productId)}}
{{replaceLink(downloadUrl)}}
```

### include
Includes a named fragment

**Syntax:** `include(fragmentName)`

**Examples:**
```
{{include "header"}}
{{include "disclaimer"}}
{{include footerFragment}}
```

## Type Conversion Functions

These functions help convert between different data types:

- `str(value)` - Convert to string
- `integer(value)` - Convert to integer
- `decimal(value)` - Convert to decimal number

## Function Chaining

Many functions can be chained together for complex operations:

```
{{uppercase(join(map("name", items), ", "))}}
// Gets all names, joins with commas, then converts to uppercase

{{format("%.2f", sum(map("price", items)))}}
// Gets all prices, sums them, then formats to 2 decimal places

{{if contains(lowercase(country), list("us", "usa", "united states"))}}
// Normalizes country name before checking
```

## Custom Functions

You can extend go-stencil with custom functions. See the [API Reference](API.md#custom-functions) for details on implementing custom functions.

## Error Handling

Functions will return an error if:
- Required arguments are missing
- Arguments are of the wrong type
- Operation fails (e.g., invalid date format)

In strict mode, these errors will cause template rendering to fail. In non-strict mode, errors are logged and a default value is used.

## Performance Notes

- Functions are evaluated during rendering, not during template preparation
- Heavy computations in functions can impact rendering performance
- Consider pre-calculating complex values in your data instead of using functions
- The `map()` function is optimized for large collections