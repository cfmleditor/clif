<cfscript>
first = new Factory({service:"A"});
second = new Factory({SERVICE:"B"});
results = {
    first = first.get("service").kind(),
    second = second.get("service").kind(),
    replaced = first.replace({service:"B"}).get("service").kind(),
    reversed = first.replace({service:"A"}).get("service").kind()
};
sharedBindings = {service:"A"};
shared = new Factory(sharedBindings);
results.beforeAliasWrite = shared.get("service").kind();
sharedBindings.service = "B";
results.afterAliasWrite = shared.get("service").kind();
registered = new Factory({});
results.registered = registered.bind("service","A").get("service").kind();
results.rebound = registered.bind("SERVICE","B").get("service").kind();
fluent = new Factory({});
results.fluent = fluent.map("service").to("A").get("service").kind();
results.fluentRebound = fluent.map("SERVICE").to("B").get("service").kind();
results.orderedRegister = fluent.register("service","A").get("service").kind();
results.forwarded = fluent.select("service").to("B").get("service").kind();
results.forwardedCollection = fluent.mapInstance("service").to("A").get("service").kind();
left = new Factory({});
right = new Factory({});
left.map("service");
right.map("other");
left.to("A");
right.to("B");
results.independentPending = left.get("service").kind();
results.engine = {name:server.coldfusion.productname,version:server.coldfusion.productversion};
writeOutput(serializeJSON(results));
</cfscript>
