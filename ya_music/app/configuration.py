from litestar.openapi import OpenAPIConfig
from litestar.openapi.plugins import SwaggerRenderPlugin

swagger = SwaggerRenderPlugin()

openapi_config = OpenAPIConfig(
    title="OpenAPI",
    version="0.1.0",
    path="/docs",
    render_plugins=[swagger],
)
