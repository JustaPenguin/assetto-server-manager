import moment = require("moment");

const nameRegex = /^[A-Za-z]{0,5}[0-9]+/;

export function prettifyName(name: string, acronyms: boolean = true): string {
    if (!name || name.length === 0) {
        return "";
    }

    let parts = name.split("_");

    if (parts[0] === "ks") {
        parts.shift();
    }

    for (let i = 0; i < parts.length; i++) {
        if ((acronyms && parts[i].length <= 3) || (acronyms && parts[i].match(nameRegex))) {
            parts[i] = parts[i].toUpperCase();
        } else {
            parts[i] = parts[i].split(' ')
                .map(w => w[0].toUpperCase() + w.substr(1).toLowerCase())
                .join(' ');
        }
    }

    return parts.join(" ")
}

export function msToTime(s: number, millisecondPrecision: boolean = true): string {
    if (!s) {
        return "";
    }

    let formatString = (millisecondPrecision ? "HH:mm:ss.SSS" : "HH:mm:ss");
    let formatted = moment.utc(s).format(formatString);

    if (formatted.startsWith("00:")) {
        // remove leading hours
        return formatted.substring(3);
    }

    return formatted;
}

function pad(num, size) {
    let s = num + "";
    while (s.length < size) {
        s = "0" + s;
    }
    return s;
}
