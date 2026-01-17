#!/usr/bin/env python3
"""
Script for converting PDF to Markdown format
"""

import fitz  # PyMuPDF
import sys

def pdf_to_markdown(pdf_path, output_path):
    """Convert PDF file to Markdown format"""
    try:
        # Открываем PDF файл
        doc = fitz.open(pdf_path)
        
        markdown_content = []
        
        print(f"Converting {len(doc)} pages...")
        
        # Обрабатываем каждую страницу
        for page_num in range(len(doc)):
            page = doc[page_num]
            
            # Извлекаем текст
            text = page.get_text()
            
            # Добавляем разделитель страниц
            markdown_content.append(f"\n---\n# Page {page_num + 1}\n\n")
            markdown_content.append(text)
            
            if (page_num + 1) % 10 == 0:
                print(f"Processed {page_num + 1} pages...")
        
        doc.close()
        
        # Сохраняем в файл
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(''.join(markdown_content))
        
        print(f"Successfully converted to {output_path}")
        return True
        
    except Exception as e:
        print(f"Error during conversion: {e}")
        return False

if __name__ == "__main__":
    if len(sys.argv) != 3:
        print("Usage: python3 pdf_to_md.py <input.pdf> <output.md>")
        sys.exit(1)
    
    pdf_path = sys.argv[1]
    output_path = sys.argv[2]
    
    success = pdf_to_markdown(pdf_path, output_path)
    sys.exit(0 if success else 1)
