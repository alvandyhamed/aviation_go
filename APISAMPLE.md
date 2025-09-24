# Template for Building List API Endpoints in Go

This guide is intended for junior Go developers who are building HTTP
API endpoints that return lists of documents from a MongoDB database. It
explains the reusable pieces of code and how to customize them for
different collections and query requirements. The examples use the Go
standard library for HTTP and the MongoDB Go driver.

## 1. Constant Building Blocks

Every list‑style endpoint shares a handful of common steps. These pieces
should be the same in each handler:

### Context with timeout

It's important to give each request a context with a short timeout so
that slow database queries do not block your server. A common pattern
is:

    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()

This creates a derived context that is cancelled after five seconds.
Always call `defer cancel()` so the context is freed when the handler
exits.

### Reading query parameters and pagination

List endpoints usually accept query parameters for search terms (`q`)
and pagination (`page`, `limit`). Helper functions like `getPage()` and
`getLimit()` can supply defaults and enforce limits. For example:

    page  := getPage(r)             // defaults to 1 if missing
    limit := getLimit(r, 20, 200)   // default 20, max 200
    skip  := int64(page-1) * limit  // number of documents to skip

`getPage()` and `getLimit()` should parse the query strings and return
sensible defaults. The `skip` variable is used later for pagination.

### Building the filter and sort

Create a `bson.M` map to store your MongoDB filter. Start empty
(`filter := bson.M{}`) and add conditions if query parameters are
provided. For example, if a `q` parameter is given you might search on
multiple fields:

    if q != "" {
        up := strings.ToUpper(q)
        filter["$or"] = []bson.M{
            {"ident": up},
            {"gps_code": up},
            {"iata_code": up},
            {"name": bson.M{"$regex": q, "$options": "i"}},
            {"municipality": bson.M{"$regex": q, "$options": "i"}},
        }
    }

For sorting, prepare a `bson.D` slice. If no specific sort is set,
define a sensible default (e.g. sort by `name` ascending):

    sort := bson.D{{Key: "name", Value: 1}} // ascending

### Creating `FindOptions`

Use `options.Find()` from the MongoDB driver to set projection, skip,
limit and sort. Projection determines which fields to return. For a
listing you usually exclude `_id` and internal fields:

    opts := options.Find().
        SetProjection(bson.M{
            "_id": 0,
            "id_csv": 0,
            "continent": 0,
            "elevation_ft": 0,
        }).
        SetSkip(skip).
        SetLimit(limit).
        SetSort(sort)

### Executing the query and reading results

Call `Find()` on the correct collection, passing the context, filter and
options. Always check for errors and close the cursor when done:

    cur, err := depMC.DB.Collection("collection_name").Find(ctx, filter, opts)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer cur.Close(ctx)

    var items []YourDTO
    if err := cur.All(ctx, &items); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

`YourDTO` is a struct type defining the fields you want to expose in
JSON responses. Create one for each collection (e.g. `AirportDTO`,
`CountryDTO`).

### Counting total documents

For pagination meta data you typically report the total number of
documents matching the filter. Use `CountDocuments`:

    total, _ := depMC.DB.Collection("collection_name").CountDocuments(ctx, filter)

Ignoring the error from `CountDocuments` is sometimes acceptable since a
count failure should not break the entire response.

### Writing the JSON response

Define a struct that includes the items slice and a `PageMeta` field
containing pagination information. Set the correct response header and
encode the struct:

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(struct {
        Items []YourDTO `json:"items"`
        Meta  PageMeta  `json:"meta"`
    }{
        Items: items,
        Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
    })

`PageMeta` is a simple struct that holds `Page`, `Limit` and `Total`
integers.

## 2. Template Endpoint

Here is a complete template combining all the pieces above. Replace
`collection_name`, `YourDTO`, the filter logic and sort order with your
own values. Add or remove query parameters as needed.

    // YourEndpoint godoc
    // @Summary      Describe your endpoint
    // @Description  Detailed description of what it does
    // @Tags         yourtag
    // @Produce      json
    // @Param        q     query  string  false  "Search term"
    // @Param        page  query  int     false  "Page number"       default(1)
    // @Param        limit query  int     false  "Items per page"    default(20)
    // @Success      200   {object}  struct{ Items []YourDTO `json:"items"`; Meta PageMeta `json:"meta"` }
    // @Failure      500   {object}  map[string]string "Internal Server Error"
    // @Router       /your-path [get]
    func YourEndpoint(w http.ResponseWriter, r *http.Request) {
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        // Read query parameters
        q    := strings.TrimSpace(r.URL.Query().Get("q"))
        page := getPage(r)
        limit := getLimit(r, 20, 200)
        skip := int64(page-1) * limit

        // Build filter
        filter := bson.M{}
        if q != "" {
            filter["field"] = bson.M{"$regex": q, "$options": "i"}
        }

        // Sort by field (ascending)
        sort := bson.D{{Key: "field", Value: 1}}

        // Projection: exclude unwanted fields
        projection := bson.M{
            "_id": 0,
        }

        // Build find options
        opts := options.Find().
            SetProjection(projection).
            SetSkip(skip).
            SetLimit(limit).
            SetSort(sort)

        // Execute query
        cur, err := depMC.DB.Collection("collection_name").Find(ctx, filter, opts)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer cur.Close(ctx)

        var items []YourDTO
        if err := cur.All(ctx, &items); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        total, _ := depMC.DB.Collection("collection_name").CountDocuments(ctx, filter)

        // Write response
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(struct {
            Items []YourDTO `json:"items"`
            Meta  PageMeta  `json:"meta"`
        }{
            Items: items,
            Meta:  PageMeta{Page: page, Limit: int(limit), Total: total},
        })
    }

### Customizing the template

When creating a new API endpoint:

1.  **Define a DTO** for the collection. Only include fields you want to
    expose to clients.
2.  **Choose a collection name** and replace `"collection_name"` with
    it.
3.  **Adjust the filter** logic based on which fields should be
    searchable. For complex filters, you can combine multiple conditions
    using `$and` and `$or`.
4.  **Set the projection** to exclude or include specific fields.
5.  **Configure sort order** by changing the `sort` variable. Use
    negative values (`-1`) for descending order.
6.  **Update Swagger annotations** (the `@Summary`, `@Description`,
    `@Tags`, `@Param` and `@Router` lines) so that the endpoint appears
    correctly in your API documentation.

By following this template, you can build clear and consistent list
endpoints while focusing only on the differences between collections.
