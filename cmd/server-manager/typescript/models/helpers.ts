export {
    ParseDate,
    ParseNumber,
    FromArray,
    ToObject,
};

// helpers
const maxUnixTSInSeconds = 9999999999;

function ParseDate(d: Date | number | string): Date {
    if (d instanceof Date) return d;
    if (typeof d === 'number') {
        if (d > maxUnixTSInSeconds) return new Date(d);
        return new Date(d * 1000); // go ts
    }
    return new Date(d);
}

function ParseNumber(v: number | string, isInt = false): number {
    if (!v) return 0;
    if (typeof v === 'number') return v;
    return (isInt ? parseInt(v) : parseFloat(v)) || 0;
}

function FromArray<T>(Ctor: { new(v: any): T }, data?: any[] | any, def = null): T[] | null {
    if (!data || !Object.keys(data).length) return def;
    const d = Array.isArray(data) ? data : [data];
    return d.map((v: any) => new Ctor(v));
}

function ToObject(o: any, typeOrCfg: any = {}, child = false): any {
    if (!o) return null;
    if (typeof o.toObject === 'function' && child) return o.toObject();

    switch (typeof o) {
        case 'string':
            return typeOrCfg === 'number' ? ParseNumber(o) : o;
        case 'boolean':
        case 'number':
            return o;
    }

    if (o instanceof Date) {
        return typeOrCfg === 'string' ? o.toISOString() : Math.floor(o.getTime() / 1000);
    }

    if (Array.isArray(o)) return o.map((v: any) => ToObject(v, typeOrCfg, true));

    const d: any = {};

    for (const k of Object.keys(o)) {
        const v: any = o[k];
        if (!v) continue;
        d[k] = ToObject(v, typeOrCfg[k] || {}, true);
    }

    return d;
}
