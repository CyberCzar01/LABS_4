import pytest
from lexer import lexer
from parser import parse
from semantic import symbol_table, func_table
from runtime import runtime_init, robot_move, robot_measure, robot_get_pos, load_labyrinth

# --------------------- Тесты лексера и парсера ------------------------

def test_lex_ident_and_keywords():
    data = "seisu x = 10; ronri y = shinri; hairetsu arr = { 5 , 6 };"
    lexer.input(data)
    tokens_list = [tok.type for tok in lexer]
    assert 'SEISU_KW' in tokens_list
    assert 'RONRI_KW' in tokens_list
    assert 'HAIRETSU_KW' in tokens_list
    assert 'IDENT' in tokens_list

def test_parse_simple_declarations():
    src = "seisu a = 5; ronri b = uso; rippotai c = { 1 , 2 , 3 , shinri };"
    ast = parse(src)
    assert symbol_table.lookup('a')['type'] == 'seisu'
    assert symbol_table.lookup('b')['type'] == 'ronri'
    assert symbol_table.lookup('c')['type'] == 'rippotai'

def test_parse_array_and_indexing():
    src = "hairetsu arr = { 4 , 5 }; seisu x; x = arr[ 3 ];"
    ast = parse(src)

def test_parse_move_measure():
    src = "{ o_0; ^_^; >_>; >_<; *_*; }"
    ast = parse(src)

# --------------------- Тесты семантики ------------------------

def test_semantic_type_error():
    with pytest.raises(Exception):
        parse("seisu x = shinri;")

def test_lvalue_type_error():
    with pytest.raises(Exception):
        parse("ronri x; x[0] = 5;")

# --------------------- Тесты runtime (move и measure) ------------------------

@pytest.fixture
def load_maze(tmp_path):
    maze_file = tmp_path / "maze_test.txt"
    maze_file.write_text(
        "3 3 1\n"
        "000\n"
        "010\n"
        "000\n"
        "1\n"
        "2 2 0\n"
        "0 0 0\n"
    )
    return str(maze_file)

def test_robot_move_and_crash(load_maze):
    runtime_init(load_maze)
    robot_move('FWD')
    robot_move('RIGHT')
    pos = robot_get_pos()
    assert pos.x == 1 and pos.y == 0 and pos.z == 0
    dist = robot_measure('FWD')
    assert dist == 0

def test_robot_measure_distance(load_maze):
    runtime_init(load_maze)
    dist = robot_measure('FWD')
    assert dist == 2

# --------------------- Тест поиска выхода ------------------------

def test_find_exit(tmp_path):
    maze_src = tmp_path / "maze_exit.txt"
    maze_src.write_text(
        "3 3 1\n"
        "000\n"
        "010\n"
        "000\n"
        "1\n"
        "2 2 0\n"
        "0 0 0\n"
    )
    script = tmp_path / "find_exit.kd"
    script.write_text(open("find_exit.kd", encoding="utf-8").read())
    ast = parse(open(str(script), 'r', encoding='utf-8').read())
    from runtime import run_program
    run_program(ast, str(maze_src))
