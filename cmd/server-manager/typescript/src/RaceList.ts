import { RRule, RRuleSet, rrulestr } from "rrule"

document.addEventListener('DOMContentLoaded', function() {
    let rRules = document.getElementsByClassName('rrule-text');

    for (let i = 0; i < rRules.length; i++) {

        let recurrenceString = rRules[i].getAttribute("data-rrule");

        if (recurrenceString) {
            let rule = RRule.fromString(recurrenceString);

            rRules[i].textContent = rule.toText()
        }
    }
});