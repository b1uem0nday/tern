do $$
    begin
        for i in 6..10 loop
                insert into test_table values(i, i*i);
            end loop;
    end;
$$;