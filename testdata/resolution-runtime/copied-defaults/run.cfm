<cfscript>
// Report observations. Engine-specific outcomes are deliberately not asserted.
function observe(required any invocation) {
    try {
        return {ok = true, result = arguments.invocation()};
    } catch (any exception) {
        return {ok = false, type = exception.type, message = exception.message};
    }
}

provider = new Provider();
host = new Host(provider);
extractedInterpolated = host.interpolated;
extractedDirect = host.direct;
results = {
    providerInterpolated = observe(function() {return provider.interpolated();}),
    providerDirect = observe(function() {return provider.direct();}),
    copiedInterpolatedOmitted = observe(function() {return host.interpolated();}),
    copiedInterpolatedSupplied = observe(function() {return host.interpolated("explicit");}),
    copiedDirectOmitted = observe(function() {return host.direct();}),
    copiedDirectSupplied = observe(function() {return host.direct("explicit");}),
    extractedInterpolated = observe(function() {return extractedInterpolated();}),
    extractedDirect = observe(function() {return extractedDirect();})
};
if (structKeyExists(server, "coldfusion")) {
    results.engine = {
        name = server.coldfusion.productname,
        version = server.coldfusion.productversion
    };
}
writeOutput(serializeJSON(results));
</cfscript>
