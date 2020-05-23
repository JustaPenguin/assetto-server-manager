import moment = require("moment");

const nameRegex = /^[A-Za-z]{0,5}[0-9]+/;

export function prettifyName(name: string, acronyms: boolean = true): string {
    try {
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
                let split = parts[i].split(' ');

                parts[i] = split.map(w => w.length > 0 ? w[0].toUpperCase() + w.substr(1).toLowerCase() : "").join(' ');
            }
        }

        return parts.join(" ");
    } catch (error) {
        return name;
    }
}

export function makeCarString(cars) {
    let out = "";

    for (let index = 0; index < cars.length; index++) {
        if (index === 0) {
            out = " - " + prettifyName(cars[index], true)
        } else {
            out = out + ", " + prettifyName(cars[index], true)
        }
    }

    return out;
}

export function msToTime(s: number, millisecondPrecision: boolean = true, trimLeadingZeroes: boolean = true): string {
    if (!s) {
        return "";
    }

    let out = "";

    if (s < 0) {
        out = "-";
        s = Math.abs(s);
    }

    let formatString = (millisecondPrecision ? "HH:mm:ss.SSS" : "HH:mm:ss");
    let formatted = moment.utc(s).format(formatString);

    if (trimLeadingZeroes && formatted.startsWith("00:")) {
        // remove leading hours
        return out + formatted.substring(3);
    }

    return out + formatted;
}

function pad(num, size) {
    let s = num + "";
    while (s.length < size) {
        s = "0" + s;
    }
    return s;
}

export function ordinalSuffix(i: number): string {
    let j = i % 10,
        k = i % 100;
    if (j === 1 && k !== 11) {
        return i + "st";
    }
    if (j === 2 && k !== 12) {
        return i + "nd";
    }
    if (j === 3 && k !== 13) {
        return i + "rd";
    }

    return i + "th";
}
