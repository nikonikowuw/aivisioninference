// Package swagger provides Swagger documentation initialization for niko-admin.
//
// The Init function is a placeholder that will be populated by `swag init`
// generating the docs package. When the docs are generated, uncomment the
// import and handler registration.
package swagger

// Init registers the Swagger UI endpoint on the given gin.Engine.
//
// Usage in router setup:
//
//	swagger.Init(r)
//
// After running `make swag`, the generated docs package should be imported:
//
//	import _ "github.com/niko-admin/niko-admin/docs"
func Init() {
	// Placeholder: after running `swag init`, uncomment below and add
	// the appropriate import for the generated docs package.
	//
	// import (
	//     swaggerFiles "github.com/swaggo/files"
	//     ginSwagger "github.com/swaggo/gin-swagger"
	//     _ "github.com/niko-admin/niko-admin/docs"
	// )
	//
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
