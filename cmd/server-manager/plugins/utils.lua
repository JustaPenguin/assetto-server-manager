local utils={};

function utils.dump(val)
    return ( dump(val) )
end

dump = function(o)
    if type(o) == 'table' then
        local s = '{ '
        for k,v in pairs(o) do
            if type(k) ~= 'number' then k = '"'..k..'"' end
            s = s .. '['..k..'] = ' .. dump(v) .. ',' .. "\n"
        end
        return s .. '} '
    else
        return tostring(o)
    end
end

return utils