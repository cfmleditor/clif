component {
    public string function primaryKey() {
        return "provider";
    }

    public string function primaryKeys() {
        return primaryKey();
    }

    public struct function interpolated(string property = "#primaryKey()#") {
        return {value = arguments.property, bodyReceiver = primaryKey()};
    }

    public struct function direct(any properties = primaryKeys()) {
        return {value = arguments.properties, bodyReceiver = primaryKey()};
    }
}
