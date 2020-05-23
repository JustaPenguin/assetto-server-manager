import $ from "jquery";
import "multiselect";
import "jquery.quicksearch";

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

const nameRegex = /^[A-Za-z]{0,5}[0-9]+/;

export function prettifyName(name: string, acronyms: boolean): string {
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


export function initMultiSelect($element: JQuery) {
    $element.each( (i: number, elem: HTMLElement): false | void => {
        let $elem = $(elem);

        if ($elem.is(":hidden")) {
            return false;
        }

        $elem.multiSelect({
            selectableHeader: "<input type='search' class='form-control search-input' autocomplete='off' placeholder='search'>",
            selectionHeader: "<input type='search' class='form-control search-input' autocomplete='off' placeholder='search'>",
            afterInit: function (ms) {
                let that = this,
                    $selectableSearch = that.$selectableUl.prev(),
                    $selectionSearch = that.$selectionUl.prev(),
                    selectableSearchString = '#' + that.$container.attr('id') + ' .ms-elem-selectable:not(.ms-selected)',
                    selectionSearchString = '#' + that.$container.attr('id') + ' .ms-elem-selection.ms-selected';

                that.qs1 = $selectableSearch.quicksearch(selectableSearchString)
                    .on('keydown', function (e) {
                        if (e.which === 40) {
                            that.$selectableUl.focus();
                            return false;
                        }
                    });

                that.qs2 = $selectionSearch.quicksearch(selectionSearchString)
                    .on('keydown', function (e) {
                        if (e.which === 40) {
                            that.$selectionUl.focus();
                            return false;
                        }
                    });
            },
            afterSelect: function () {
                this.qs1.cache();
                this.qs2.cache();
            },
            afterDeselect: function () {
                this.qs1.cache();
                this.qs2.cache();
            }
        });
    });
}


export function msToTime(s) {
    // Pad to 2 or 3 digits, default is 2
    let pad = (n, z = 2) => ('00' + n).slice(-z);
    return pad(s / 3.6e6 | 0) + ':' + pad((s % 3.6e6) / 6e4 | 0) + ':' + pad((s % 6e4) / 1000 | 0) + '.' + pad(s % 1000, 3);
}

export function timeDiff(tstart: Date, tend: Date) {
    let diff = Math.floor((tend.getTime() - tstart.getTime()) / 1000), units = [
        {d: 60, l: "s"},
        {d: 60, l: "m"},
        {d: 24, l: "hr"},
    ];

    let s = '';
    for (let i = 0; i < units.length; i++) {
        if (diff === 0) {
            continue
        }
        s = (diff % units[i].d) + units[i].l + " " + s;
        diff = Math.floor(diff / units[i].d);
    }
    return s;
}
