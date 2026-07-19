#!/bin/bash
set -euo pipefail
python3 - <<'EOF'
from pathlib import Path

def tokenize(s):
    i=0; out=[]
    while i<len(s):
        c=s[i]
        if c.isspace():
            i+=1; continue
        if c.isalpha() or c=='_':
            j=i+1
            while j<len(s) and (s[j].isalnum() or s[j]=='_'):
                j+=1
            out.append(('id', s[i:j])); i=j; continue
        if c.isdigit():
            j=i+1
            while j<len(s) and s[j].isdigit():
                j+=1
            out.append(('num', int(s[i:j]))); i=j; continue
        if c in '+-*/()':
            out.append(('op', c)); i+=1; continue
        raise ValueError(c)
    return out

def eval_expr(expr, env):
    tokens=tokenize(expr)
    fixed=[]; prev=None
    for t in tokens:
        if t[0]=='op' and t[1]=='-' and (prev is None or (prev[0]=='op' and prev[1] != ')')):
            fixed.append(('op','u-'))
        else:
            fixed.append(t)
        prev=t
    prec={'u-':3,'*':2,'/':2,'+':1,'-':1}
    right={'u-'}
    outq=[]; st=[]
    for t in fixed:
        if t[0] in ('num','id'):
            outq.append(t)
        elif t[1]=='(':
            st.append(t)
        elif t[1]==')':
            while st and st[-1][1] != '(':
                outq.append(st.pop())
            st.pop()
        else:
            while st and st[-1][1] != '(':
                top=st[-1][1]
                if top in prec and ((top not in right and prec[top] >= prec[t[1]]) or (top in right and prec[top] > prec[t[1]])):
                    outq.append(st.pop())
                else:
                    break
            st.append(t)
    while st:
        outq.append(st.pop())
    st=[]
    for t in outq:
        if t[0]=='num':
            st.append(t[1])
        elif t[0]=='id':
            st.append(env[t[1]])
        elif t[1]=='u-':
            st.append(-st.pop())
        else:
            b=st.pop(); a=st.pop()
            if t[1]=='+': st.append(a+b)
            elif t[1]=='-': st.append(a-b)
            elif t[1]=='*': st.append(a*b)
            elif t[1]=='/': st.append(int(a/b))
    return st[0]

env={}
for line in Path('/app/vars.env').read_text().splitlines():
    line=line.strip()
    if not line:
        continue
    k,v=line.split('=',1)
    env[k.strip()]=int(v.strip())
vals=[]
for line in Path('/app/exprs.txt').read_text().splitlines():
    if not line.strip():
        continue
    vals.append(str(eval_expr(line, env)))
Path('/app/values.txt').write_text('\n'.join(vals)+'\n')
EOF
