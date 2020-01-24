local utils={};

function utils.dump(val)
    return ( dump(val, 1) )
end

dump = function(o, d)
    if type(o) == 'table' then
        local t = ''
        local t1 = ''

        for i = 1,d do
           t = t .. "\t"
        end

        for i = 1,d-1 do
            t1 = t1 .. "\t"
        end

        local s = '{ ' .. "\n"
        for k,v in pairs(o) do
            if type(k) ~= 'number' then k = '"'..k..'"' end
            s = s .. t .. '['..k..'] = ' .. dump(v, d+1) .. ',' .. "\n"
        end
        return s .. "\n" .. t1 .. '} '
    else
        return tostring(o)
    end
end

-- Open JSON file
function utils.jsonOpen(location, filename)
    local path = (location .."/".. filename)
    local f = assert(io.open(path, "r"))
    local result = f:read "*a"

    f:close()
    return result
end

return utils