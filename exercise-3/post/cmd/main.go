package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type BookStore struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	BookName   string
	BookAuthor string
	BookISBN   string
	BookPages  int
	BookYear   int
}

type PostBookDTO struct {
	Name   string `json:"name"`
	Author string `json:"author"`
	Pages  int    `json:"pages"`
	Year   int    `json:"year"`
	Isbn   string `json:"isbn,omitempty"`
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			BookName:   "The Vortex",
			BookAuthor: "José Eustasio Rivera",
			BookISBN:   "958-30-0804-4",
			BookPages:  292,
			BookYear:   1924,
		},
		{
			BookName:   "Frankenstein",
			BookAuthor: "Mary Shelley",
			BookISBN:   "978-3-649-64609-9",
			BookPages:  280,
			BookYear:   1818,
		},
		{
			BookName:   "The Black Cat",
			BookAuthor: "Edgar Allan Poe",
			BookISBN:   "978-3-99168-238-7",
			BookPages:  280,
			BookYear:   1843,
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":         res.ID.Hex(),
			"BookName":   res.BookName,
			"BookAuthor": res.BookAuthor,
			"BookISBN":   res.BookISBN,
			"BookPages":  res.BookPages,
			"BookYear":   res.BookYear,
		})
	}

	return ret
}

type BookDTO struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Author string `json:"author"`
	Pages  int    `json:"pages"`
	Year   int    `json:"year"`
	Isbn   string `json:"isbn,omitempty"`
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// database connection initialization
	uri := os.Getenv("DATABASE_URI")
	if len(uri) == 0 {
		fmt.Printf("failure to load env variable\n")
		os.Exit(1)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		fmt.Printf("failed to create client for MongoDB\n")
		os.Exit(1)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		fmt.Printf("failed to connect to MongoDB, please make sure the database is running\n")
		os.Exit(1)
	}

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()
	coll, err := prepareDatabase(client, "exercise-1", "information")
	prepareData(client, coll)
	e := echo.New()
	e.Use(middleware.Logger())

	e.POST("/api/books", func(c echo.Context) error {
		book := new(PostBookDTO)
		err = c.Bind(book)
		if err != nil {
			fmt.Println("error in conversion", err)
			return c.JSON(http.StatusNotModified, "error in payload conversion ")
		}

		// create field to compare
		objToComapare := bson.M{}
		if book.Name != "" {
			objToComapare["bookname"] = book.Name
		}
		if book.Author != "" {
			objToComapare["bookauthor"] = book.Author
		}
		if book.Pages != 0 {
			objToComapare["bookpages"] = book.Pages
		}
		if book.Year != 0 {
			objToComapare["bookyear"] = book.Year
		}
		if book.Isbn != "" {
			objToComapare["bookisbn"] = book.Isbn
		}

		// check object existence
		var existingBook BookStore
		found := coll.FindOne(context.TODO(), objToComapare).Decode(&existingBook)
		if found == nil {
			return c.JSON(http.StatusNotModified, book)
		}

		bookStore := BookStore{
			BookName:   book.Name,
			BookAuthor: book.Author,
			BookPages:  book.Pages,
			BookYear:   book.Year,
			BookISBN:   book.Isbn,
		}
		result, err := coll.InsertOne(context.TODO(), bookStore)
		if err != nil {
			return c.JSON(http.StatusNotModified, "invalid on insertion")
		}
		bookId := result.InsertedID.(primitive.ObjectID)
		insertedIDString := bookId.Hex()

		payload := BookDTO{
			Id:     insertedIDString,
			Name:   book.Name,
			Author: book.Author,
			Pages:  book.Pages,
			Year:   book.Year,
			Isbn:   book.Isbn,
		}
		return c.JSON(http.StatusOK, payload)
	})

	e.Logger.Fatal(e.Start(":3032"))
}
