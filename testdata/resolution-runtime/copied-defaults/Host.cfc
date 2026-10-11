component {
    public any function init(required any provider) {
        variables.interpolated = arguments.provider.interpolated;
        this.interpolated = arguments.provider.interpolated;
        variables.direct = arguments.provider.direct;
        this.direct = arguments.provider.direct;
        return this;
    }

    public string function primaryKey() {
        return "host";
    }

    public string function primaryKeys() {
        return primaryKey();
    }
}
