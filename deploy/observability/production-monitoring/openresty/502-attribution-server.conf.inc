# Add inside the HTTPS server block.
access_log /www/sites/alltokenapi.com/log/gateway-5xx.log alltoken_5xx if=$alltoken_log_5xx;
error_page 502 = @alltoken_gateway_502;

# Add inside the proxied application location. Upstream 502 responses are kept
# intact; only OpenResty-generated 502 responses use the named location below.
proxy_intercept_errors off;
add_header X-Alltoken-Error-Layer $alltoken_502_layer always;
add_header X-Alltoken-Error-Type $alltoken_502_type always;

# Add inside the HTTPS server block alongside the application location.
location @alltoken_gateway_502 {
    internal;
    default_type application/json;
    add_header X-Alltoken-Error-Layer gateway always;
    add_header X-Alltoken-Error-Type gateway_no_application_response always;
    return 502 '{"error":{"message":"AllToken gateway could not get a response from an application instance","type":"gateway_error","code":"gateway_no_application_response","source":"alltoken"}}';
}

# Deterministic local-only probe for validating the gateway classification.
location = /__gateway_502_probe {
    allow 127.0.0.1;
    allow ::1;
    deny all;
    return 502;
}
