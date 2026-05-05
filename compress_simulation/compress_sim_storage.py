"""
Author: Joon Sung Park (joonspk@stanford.edu)

File: compress_sim_storage.py
Description: Compresses a simulation for replay demos. 
"""
import shutil
import json
from global_methods import *


def compress(sim_code):
    sim_storage = f"../frontend_server/storage/{sim_code}"
    compressed_storage = f"../frontend_server/compressed_storage/{sim_code}"
    persona_folder = sim_storage + "/personas"
    move_folder = sim_storage + "/movement"
    meta_file = sim_storage + "/reverie/meta.json"

    persona_names = []
    for i in find_filenames(persona_folder, ""):
        x = i.split("/")[-1].strip()
        if x[0] != ".":
            persona_names += [x]

    move_steps = [int(i.split("/")[-1].split(".")[0])
                  for i in find_filenames(move_folder, "json")]
    min_move_count = min(move_steps)
    max_move_count = max(move_steps)

    persona_last_move = dict()
    master_move = dict()
    for i in range(min_move_count, max_move_count+1):
        master_move[i - min_move_count] = dict()
        with open(f"{move_folder}/{str(i)}.json") as json_file:
            i_move_dict = json.load(json_file)["persona"]
            for p in persona_names:
                move = False
                if i == min_move_count:
                    move = True
                elif (i_move_dict[p]["movement"] != persona_last_move[p]["movement"]
                      or i_move_dict[p]["pronunciato"] != persona_last_move[p]["pronunciatio"]
                      or i_move_dict[p]["description"] != persona_last_move[p]["description"]
                      or i_move_dict[p]["chat"] != persona_last_move[p]["chat"]):
                    move = True

                if move:
                    persona_last_move[p] = {"movement": i_move_dict[p]["movement"],
                                            "pronunciatio": i_move_dict[p]["pronunciato"],
                                            "description": i_move_dict[p]["description"],
                                            "chat": i_move_dict[p]["chat"]}
                    master_move[i - min_move_count][p] = {"movement": i_move_dict[p]["movement"],
                                                          "pronunciatio": i_move_dict[p]["pronunciato"],
                                                          "description": i_move_dict[p]["description"],
                                                          "chat": i_move_dict[p]["chat"]}

    create_folder_if_not_there(compressed_storage)
    with open(f"{compressed_storage}/master_movement.json", "w") as outfile:
        outfile.write(json.dumps(master_move, indent=2))

    shutil.copyfile(meta_file, f"{compressed_storage}/meta.json")
    shutil.copytree(persona_folder, f"{compressed_storage}/personas/")


if __name__ == '__main__':
    compress("base_cafe_spiral")
