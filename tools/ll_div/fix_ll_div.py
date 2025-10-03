#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import re
import sys

def parse_unicode_blocks(unicode_file):
    """
    从Unicode.txt文件中解析Unicode区段信息
    返回一个包含(起始编码, 结束编码, 区段名称)的列表
    """
    blocks = []
    
    with open(unicode_file, 'r', encoding='utf-8') as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
                
            # 解析格式: "0 BMP	U+2E80..U+2EFF	中日韩汉字部首补充"
            parts = line.split('\t')
            if len(parts) >= 3:
                range_part = parts[1]
                block_name = parts[2]
                
                # 解析Unicode范围
                range_match = re.match(r'U\+([0-9A-Fa-f]+)\.\.U\+([0-9A-Fa-f]+)', range_part)
                if range_match:
                    start_hex = range_match.group(1)
                    end_hex = range_match.group(2)
                    
                    start_code = int(start_hex, 16)
                    end_code = int(end_hex, 16)
                    
                    blocks.append((start_code, end_code, block_name))
    
    # 按起始编码排序，便于二分查找
    blocks.sort(key=lambda x: x[0])
    return blocks

def get_unicode_block_improved(unicode_point, blocks):
    """
    根据Unicode编码点和区段列表返回Unicode区段别名
    """
    code_point = int(unicode_point, 16)
    
    # 使用二分查找确定编码点所在的区段
    left, right = 0, len(blocks) - 1
    
    while left <= right:
        mid = (left + right) // 2
        start_code, end_code, block_name = blocks[mid]
        
        if start_code <= code_point <= end_code:
            return block_name
        elif code_point < start_code:
            right = mid - 1
        else:
            left = mid + 1
    
    # 如果没有找到匹配的区段，返回"其他區段"
    return "其他區段"

def fix_ll_div_file_improved(input_file, unicode_file, output_file=None):
    """
    修复ll_div.txt文件中的#N/A项，使用Unicode.txt中的区段定义
    """
    if output_file is None:
        output_file = input_file + ".improved"
    
    # 解析Unicode区段信息
    print("正在解析Unicode区段信息...")
    blocks = parse_unicode_blocks(unicode_file)
    print(f"已加载 {len(blocks)} 个Unicode区段")
    
    fixed_count = 0
    
    with open(input_file, 'r', encoding='utf-8') as f_in, \
         open(output_file, 'w', encoding='utf-8') as f_out:
        
        for line_num, line in enumerate(f_in, 1):
            line = line.rstrip('\n')
            
            # 检查是否包含#N/A
            if '#N/A' in line:
                # 解析行内容
                parts = line.split('\t')
                if len(parts) != 2:
                    print(f"警告: 第{line_num}行格式异常: {line}")
                    f_out.write(line + '\n')
                    continue
                
                character = parts[0]
                info_str = parts[1]
                
                # 解析信息部分 [部件拆分,拼音,Unicode区段别名,Unicode编码]
                match = re.match(r'\[([^,]*),([^,]*),([^,]*),([^,]*)\]', info_str)
                if not match:
                    print(f"警告: 第{line_num}行信息格式异常: {info_str}")
                    f_out.write(line + '\n')
                    continue
                
                component, pinyin, block_alias, unicode_code = match.groups()
                
                # 如果Unicode编码是#N/A，则生成
                if unicode_code == '#N/A':
                    # 获取字符的Unicode编码
                    if character:
                        unicode_hex = hex(ord(character))[2:].upper()
                        unicode_code = f"U+{unicode_hex}"
                        
                        # 根据Unicode编码生成区段别名
                        block_alias = get_unicode_block_improved(unicode_hex, blocks)
                        
                        # 构建新的信息字符串
                        new_info = f"[{component},{pinyin},{block_alias},{unicode_code}]"
                        new_line = f"{character}\t{new_info}"
                        
                        f_out.write(new_line + '\n')
                        fixed_count += 1
                        
                        if fixed_count <= 10:  # 只显示前10个修复示例
                            print(f"修复第{line_num}行: {character} -> {unicode_code}, {block_alias}")
                    else:
                        print(f"警告: 第{line_num}行字符为空")
                        f_out.write(line + '\n')
                else:
                    # 如果Unicode编码不是#N/A，但区段别名是#N/A
                    if block_alias == '#N/A' and unicode_code != '#N/A':
                        # 从Unicode编码中提取编码点
                        unicode_match = re.match(r'U\+([0-9A-Fa-f]+)', unicode_code)
                        if unicode_match:
                            unicode_hex = unicode_match.group(1)
                            block_alias = get_unicode_block_improved(unicode_hex, blocks)
                            
                            # 构建新的信息字符串
                            new_info = f"[{component},{pinyin},{block_alias},{unicode_code}]"
                            new_line = f"{character}\t{new_info}"
                            
                            f_out.write(new_line + '\n')
                            fixed_count += 1
                            
                            if fixed_count <= 10:  # 只显示前10个修复示例
                                print(f"修复第{line_num}行: {character} -> {block_alias}")
                        else:
                            f_out.write(line + '\n')
                    else:
                        f_out.write(line + '\n')
            else:
                # 没有#N/A的行直接写入
                f_out.write(line + '\n')
    
    print(f"\n修复完成! 共修复 {fixed_count} 行")
    print(f"输出文件: {output_file}")
    
    return fixed_count

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("用法: python fix_ll_div_improved.py <input_file> <unicode_file> [output_file]")
        print("示例: python fix_ll_div_improved.py ll_div.txt Unicode.txt ll_div_improved.txt")
        sys.exit(1)
    
    input_file = sys.argv[1]
    unicode_file = sys.argv[2]
    output_file = sys.argv[3] if len(sys.argv) > 3 else None
    
    try:
        fixed_count = fix_ll_div_file_improved(input_file, unicode_file, output_file)
        if fixed_count > 0:
            print(f"成功修复了 {fixed_count} 个#N/A项")
        else:
            print("未找到需要修复的#N/A项")
    except Exception as e:
        print(f"处理文件时出错: {e}")
        sys.exit(1)