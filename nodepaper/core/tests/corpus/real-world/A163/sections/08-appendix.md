# 附录

支撑材料的文件列表：

1. result1.xlsx
2. result2.xlsx
3. result4.xlsx
4. function.py
5. zeropoint.py
6. number.py
7. 问题一代码.py
8. crushjudge.py
9. 问题二代码.py
10. 问题三代码.py
11. dataproduce.py
12. positioniteration.py
13. velocityiteration.py
14. 问题四代码.py
15. velctorytheta.py
16. 问题五代码.py

代码：

## function

```python
import numpy as np
def f1 ( theta ):
result = theta *np . sqrt ( theta ** 2+1 )+np . log ( theta +np . sqrt ( theta ** 2+
1 ) )
return result
#用于计算龙头位置随时间的变化
def f2 ( theta , theta0 , v0 ,t , d ):
result =f1 ( theta0 )-f1 ( theta )-4*v0*t*np .pi/d
return result
#用于计算龙头位置随时间的变化
def f3 ( theta ,d , d0 , theta_last ):
t= theta
t_l= theta_last

result =t ** 2+ t_l ** 2-2*t* t_l *np .cos ( t-t_l )-4*np .pi ** 2*d0 ** 2/d ** 2
return result
#用于计算盘入螺线上的位置迭代
def f4 ( theta ):
t= theta
result =( np . sin( t )+t*np . cos ( t ) )/( np .cos ( t )-t*np .sin ( t ) )
return result
#用于计算螺线上速度的斜率
def f5 ( theta ,d , d0 , theta_last ):
t= theta +np .pi
t_l= theta_last +np .pi
result =t ** 2+ t_l ** 2-2*t* t_l *np .cos ( t-t_l )-4*np .pi ** 2*d0 ** 2/d ** 2
return result
#用于计算盘出螺线上的位置迭代
def f6 ( theta ,d , d0 , theta0 ,l , gamma ):
t= theta
t0= theta0
result =l ** 2+d ** 2*t ** 2/( 4*np .pi ** 2 )-d*l*t*np .cos ( t-t0+ gamma )/np .
pi-d0 ** 2
return result
#用于计算前把手在第一段圆弧而后把手在盘入螺线的情形
import numpy as np
def f1 ( theta ):
result = theta *np . sqrt ( theta ** 2+1 )+np . log ( theta +np . sqrt ( theta ** 2+
1 ) )
return result
#用于计算龙头位置随时间的变化
def f2 ( theta , theta0 , v0 ,t , d ):
result =f1 ( theta0 )-f1 ( theta )-4*v0*t*np .pi/d
return result
#用于计算龙头位置随时间的变化
def f3 ( theta ,d , d0 , theta_last ):
t= theta
t_l= theta_last
result =t ** 2+ t_l ** 2-2*t* t_l *np .cos ( t-t_l )-4*np .pi ** 2*d0 ** 2/d ** 2
return result
#用于计算盘入螺线上的位置迭代
def f4 ( theta ):
t= theta
result =( np . sin( t )+t*np . cos ( t ) )/( np .cos ( t )-t*np .sin ( t ) )
return result
#用于计算螺线上速度的斜率
def f5 ( theta ,d , d0 , theta_last ):
t= theta +np .pi
t_l= theta_last +np .pi
result =t ** 2+ t_l ** 2-2*t* t_l *np .cos ( t-t_l )-4*np .pi ** 2*d0 ** 2/d ** 2
return result
#用于计算盘出螺线上的位置迭代
def f6 ( theta ,d , d0 , theta0 ,l , gamma ):
t= theta
t0= theta0
result =l ** 2+d ** 2*t ** 2/( 4*np .pi ** 2 )-d*l*t*np .cos ( t-t0+ gamma )/np .
pi-d0 ** 2
return result
#用于计算前把手在第一段圆弧而后把手在盘入螺线的情形
```

## zeropoint

```python
def zero1 (f ,a ,b ,e , theta0 , v0 ,t , d ):
while b-a>=e:
c=( a+b )/2 #取中点
if f (a , theta0 , v0 ,t , d )*f (c , theta0 , v0 ,t , d )<0:
b=c
else :
a=c
#判断零点所在区间
return ( a+b )/2
#用于计算函数f2的零点
def zero2 (f ,a ,b ,e ,d , d0 , theta_last ):
while b-a>=e:
c=( a+b )/2 #取中点
if f (a ,d , d0 , theta_last )*f (c ,d , d0 , theta_last )<0:
b=c
else :
a=c
#判断零点所在区间
return ( a+b )/2
#用于计算函数f3和f5的零点
def zero3 (f ,a ,b ,e ,d , d0 , theta0 ,l , gamma ):
while b-a>=e:
c=( a+b )/2 #取中点
if f (a ,d , d0 , theta0 ,l , gamma )*f (c ,d , d0 , theta0 ,l , gamma )<0:
b=c
else :
a=c
#判断零点所在区间
return ( a+b )/2
#用于计算函数f6的零点
```

## number

```python
import numpy as np
def number (A , n ):
for i in np . arange ( A . shape [0]):
for j in np . arange ( A . shape [1]):
a=A[i , j]
b=int( a*10 ** n )*10 ** (-n )
#获得a的前n位小数
if a-b>=5*10 ** (-n-1 ):
b=b+10 ** (-n )
#四舍五入
A[i , j]=b
return A
#用于保留6位小数
```

## 问题一代码

```python
import numpy as np
import pandas as pd
from function import f2 #用于计算龙头位置随时间的变化
from function import f3 #用于计算盘入螺线上的位置迭代
from function import f4 #用于计算螺线上速度的斜率
from zeropoint import zero1 #用于计算函数f2的零点
from zeropoint import zero2 #用于计算函数f3的零点
from number import number #用于保留6位小数

d=0 . 55 #螺距
v0=1 #龙头速度
theta0 =32*np .pi #龙头初始极角
lst_chair_theta =[]
for t in np . arange ( 301 ):
if t==0:
theta_chair0 = theta0
else :
theta_chair0 = zero1 ( f2 ,0 , theta0 , 10 ** (-8 ) , theta0 , v0 ,t , d )
#给出龙头的t极角theta
lst_theta =[ theta_chair0 ]
for i in np . arange ( 223 ):
if i==0:
d0=3 . 41-0 . 275 *2
else :
d0=2 . 2-0 . 275 *2
#确定板长
theta_last = lst_theta [-1] #获得上一个把手的极角theta
theta = zero2 ( f3 , theta_last , theta_last +np .pi/2 , 10 ** (-8 ) ,d , d0 ,
theta_last )
lst_theta . append ( theta )
#计算当前把手的极角theta
lst_chair_theta . append ( lst_theta )
lst_chair_theta =np . array ( lst_chair_theta )
lst_chair_xy =[]
for t in np . arange ( 301 ):
lst_xy =[]
for i in np . arange ( 224 ):
theta = lst_chair_theta [t , i]
lst_xy . append ( d* theta *np .cos ( theta )/( 2*np .pi) )
lst_xy . append ( d* theta *np .sin ( theta )/( 2*np .pi) )
#根据theta角度确定把手坐标（x，y）
lst_chair_xy . append ( lst_xy )
lst_chair_xy =np . array ( lst_chair_xy ) . T
lst_chair_xy = number ( lst_chair_xy , 6 ) #保留6位小数
df=pd . DataFrame ( lst_chair_xy )
df . to_excel (" result1_1 . xlsx ", index = False )
#保存数据到Excel中
lst_chair_v =[]
for t in np . arange (0 , 301 ):
lst_v =[v0]
for i in np . arange ( 223 ):
v_last = lst_v [-1] #获得上一个把手的速度
theta_last = lst_chair_theta [t , i]
theta = lst_chair_theta [t , i+1]
x_last = lst_chair_xy [i*2 , t]
y_last = lst_chair_xy [i*2+1 , t]
x= lst_chair_xy [i*2+2 , t]
y= lst_chair_xy [i*2+3 , t]
#获得上一个把手和当前把手的坐标（x，y）和极角
k_chair =( y_last -y )/( x_last -x )
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算板凳和两个速度的斜率
aleph1 =np . arctan ( np .abs (( k_v_last - k_chair )/( 1+ k_v_last *
k_chair ) ) )
aleph2 =np . arctan ( np .abs (( k_v - k_chair )/( 1+k_v * k_chair ) ) )
#计算两个速度与板凳的夹角

v= v_last *np .cos( aleph1 )/np .cos( aleph2 ) #计算当前把手的速度
lst_v . append ( v )
lst_chair_v . append ( lst_v )
lst_chair_v =np . array ( lst_chair_v ) . T
lst_chair_v = number ( lst_chair_v , 6 ) #保留6位小数
df=pd . DataFrame ( lst_chair_v )
df . to_excel (" result1_2 . xlsx ", index = False )
#保存数据到Excel中
```

## crushjudge

```python
import numpy as np
from function import f3 #用于计算盘入螺线上的位置迭代
from zeropoint import zero2 #用于计算函数f3的零点
def judge ( theta ,d , v0 ):
d1=0 . 275
d2=0 . 15
lst_theta =[ theta ]
for i in np . arange ( 223 ):
if i==0:
d0=3 . 41-0 . 275 *2
else :
d0=2 . 2-0 . 275 *2
#确定板长
theta_last = lst_theta [-1] #获得上一个把手的极角theta
a= theta_last
b= theta_last +np .pi/2
theta_new = zero2 ( f3 ,a ,b , 10 ** (-8 ) ,d , d0 , theta_last )
lst_theta . append ( theta_new )
#计算当前把手的极角theta
if theta_new - theta >=3*np .pi:
break
#当当前把手位置离龙头过远时结束
lst_x =[]
lst_y =[]
for i in np . arange (len( lst_theta ) ):
p= lst_theta [i]*d/( 2*np .pi)
x=p*np .cos( lst_theta [i])
y=p*np .sin( lst_theta [i])
lst_x . append ( x )
lst_y . append ( y )
#根据theta角度确定把手坐标（x，y）
lst_k =[]
for i in np . arange (len( lst_theta )-1 ):
k=( lst_y [i]- lst_y [i+1])/( lst_x [i]- lst_x [i+1])
lst_k . append ( k )
#计算板凳的斜率
k1= lst_k [0]
x1= lst_x [0]
y1= lst_y [0]
#获得龙头的坐标和斜率
k2=( d2/d1+k1 )/( 1-d2*k1/d1 ) #计算龙头前把手和外前点直线的斜率
b=d2*np . sqrt ( k1 ** 2+1 )+y1-k1*x1
if np .abs ( b )<=np .abs ( y1-k1*x1 ):
b=-d2*np . sqrt ( k1 ** 2+1 )+y1-k1*x1
#计算龙头外侧边的截距
x=( y1-k2*x1-b )/( k1-k2 )
y=( k1*y1-k1*k2*x1-k2*b )/( k1-k2 )

#计算龙头外前点的坐标
flag =0
for i in np . arange (len( lst_k ) ):
if lst_theta [i+1]- theta >=np .pi:
ki= lst_k [i]
xi= lst_x [i]
yi= lst_y [i]
d_chair =np .abs ( ki*( x-xi )+yi-y )/np . sqrt ( ki ** 2+1 )
#计算龙头外前点到当前板凳中心线的距离
if d_chair <d2:
flag =1
#判断是否相撞
x2= lst_x [1]
y2= lst_y [1]
#获得第一节龙身前把手的坐标
k2=( k1-d2/d1 )/( 1+d2*k1/d1 ) #计算第一节龙身前把手和龙头外后点直线的斜率
x=( y2-k2*x2-b )/( k1-k2 )
y=( k1*y2-k1*k2*x2-b*k2 )/( k1-k2 )
#计算龙头外后点的坐标
for i in np . arange (len( lst_k ) ):
if lst_theta [i+1]- theta >=np .pi:
ki= lst_k [i]
xi= lst_x [i]
yi= lst_y [i]
d_chair =np .abs ( ki*( x-xi )+yi-y )/np . sqrt ( ki ** 2+1 )
#计算龙头外后点到当前板凳中心线的距离
if d_chair <d2:
flag =1
#判断是否相撞
k1= lst_k [1]
x1= lst_x [1]
y1= lst_y [1]
#获得第一节龙身的前把手坐标和斜率
k2=( d2/d1+k1 )/( 1-d2*k1/d1 ) #计算第一节龙身前把手和外前点直线的斜率
b=d2*np . sqrt ( k1 ** 2+1 )+y1-k1*x1
if np .abs ( b )<=np .abs ( y1-k1*x1 ):
b=-d2*np . sqrt ( k1 ** 2+1 )+y1-k1*x1
#计算第一节龙身外侧边的截距
x=( y1-k2*x1-b )/( k1-k2 )
y=( k1*y1-k1*k2*x1-k2*b )/( k1-k2 )
#计算第一节龙身外前点的坐标
for i in np . arange (len( lst_k ) ):
if lst_theta [i+1]- theta >=np .pi:
ki= lst_k [i]
xi= lst_x [i]
yi= lst_y [i]
d_chair =np .abs ( ki*( x-xi )+yi-y )/np . sqrt ( ki ** 2+1 )
#计算第一节龙身外前点到当前板凳中心线的距离
if d_chair <d2:
flag =1
#判断是否相撞
x2= lst_x [2]
y2= lst_y [2]
#获得第二节龙身前把手的坐标
k2=( k1-d2/d1 )/( 1+d2*k1/d1 ) #计算第二节龙身前把手和第一节龙身外后点直线的
斜率
x=( y2-k2*x2-b )/( k1-k2 )
y=( k1*y2-k1*k2*x2-b*k2 )/( k1-k2 )

#计算第一节龙身外后点的坐标
for i in np . arange (len( lst_k ) ):
if lst_theta [i+1]- theta >=np .pi:
ki= lst_k [i]
xi= lst_x [i]
yi= lst_y [i]
d_chair =np .abs ( ki*( x-xi )+yi-y )/np . sqrt ( ki ** 2+1 )
#计算第一节龙身外后点到当前板凳中心线的距离
if d_chair <d2:
flag =1
#判断是否相撞
return flag
#用于判断是否发生碰撞
```

## 问题二代码

```python
import numpy as np
import pandas as pd
from function import f1 #用于计算龙头位置随时间的变化
from function import f3 #用于计算盘入螺线上的位置迭代
from function import f4 #用于计算螺线上速度的斜率
from zeropoint import zero2 #用于计算函数f3的零点
from number import number #用于保留6位小数
from crashjudge import judge #用于判断是否发生碰撞
d=0 . 55 #螺距
v0=1 #龙头速度
theta0 =32*np .pi #龙头初始极角
for theta in np . arange ( 60 ,0 ,-0 . 01 ):
flag = judge ( theta ,d , v0 )
if flag :
break
#判断龙头处于当前位置时是否有板凳碰撞
for theta in np . arange ( theta +0 . 01 , theta -0 . 01 ,-0 . 0001 ):
flag = judge ( theta ,d , v0 )
if flag :
break
for theta in np . arange ( theta +0 . 0001 , theta -0 . 0001 ,-0 . 000001 ):
flag = judge ( theta ,d , v0 )
if flag :
break
#细化碰撞时龙头的极角theta
theta_chair0 = theta +0 . 000001
t=d*( f1 ( theta0 )-f1 ( theta ) )/( 4*np .pi*v0 ) #计算碰撞的时刻
lst_chair_theta =[ theta_chair0 ]
for i in np . arange ( 223 ):
if i==0:
d0=3 . 41-0 . 275 *2
else :
d0=2 . 2-0 . 275 *2
#确定板长
theta_last = lst_chair_theta [-1] #获得上一个把手的极角theta
theta = zero2 ( f3 , theta_last , theta_last +np .pi/2 , 10 ** (-8 ) ,d , d0 ,
theta_last )
lst_chair_theta . append ( theta )
#计算当前把手的极角theta
lst_chair_xyv =[]
for i in np . arange ( 224 ):
lst_xyv =[]

theta = lst_chair_theta [i]
lst_xyv . append ( d* theta *np .cos ( theta )/( 2*np .pi) )
lst_xyv . append ( d* theta *np .sin ( theta )/( 2*np .pi) )
#根据theta角度确定把手坐标（x，y）
lst_xyv . append ( v0 )
lst_chair_xyv . append ( lst_xyv )
lst_chair_xyv =np . array ( lst_chair_xyv )
for i in np . arange ( 223 ):
v_last = lst_chair_xyv [i , 2] #获得上一个把手的速度
theta_last = lst_chair_theta [i]
theta = lst_chair_theta [i+1]
x_last = lst_chair_xyv [i , 0]
y_last = lst_chair_xyv [i , 1]
x= lst_chair_xyv [i+1 , 0]
y= lst_chair_xyv [i+1 , 1]
#获得上一个把手和当前把手的坐标（x，y）和极角
k_chair =( y_last -y )/( x_last -x )
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算板凳和两个速度的斜率
aleph1 =np . arctan ( np . abs (( k_v_last - k_chair )/( 1+ k_v_last * k_chair )
) )
aleph2 =np . arctan ( np . abs (( k_v - k_chair )/( 1+k_v * k_chair ) ) )
#计算两个速度与板凳的夹角
v= v_last *np .cos( aleph1 )/np .cos( aleph2 ) #计算当前把手的速度
lst_chair_xyv [i+1 , 2]=v
lst_chair_xyv = number ( lst_chair_xyv , 6 ) #保留6位小数
df=pd . DataFrame ( lst_chair_xyv )
df . to_excel (" result2_ . xlsx ", index = False )
#保存数据到Excel中
```

## 问题三代码

```python
import numpy as np
from crashjudge import judge #用于判断是否发生碰撞
v0=1 #龙头速度
D=9 #调头空间的直径
for d in np . arange ( 0 . 55 , 0 .4 ,-0 . 01 ):
theta_min =D*np .pi/d #确定进入调头空间时的极角theta
for theta in np . arange ( theta_min +6 , theta_min ,-0 . 1 ):
flag = judge ( theta ,d , v0 )
if flag :
break
if flag :
break
#判断当前螺距是否在调头空间外有板凳碰撞
for d in np . arange ( d+0 . 01 , d-0 . 01 ,-0 . 0001 ):
theta_min =D*np .pi/d
for theta in np . arange ( theta_min +6 , theta_min ,-0 . 1 ):
flag = judge ( theta ,d , v0 )
if flag :
break
if flag :
break
for d in np . arange ( d+0 . 0001 , d-0 . 0001 ,-0 . 00001 ):
theta_min =D*np .pi/d
for theta in np . arange ( theta_min +6 , theta_min ,-0 . 1 ):
flag = judge ( theta ,d , v0 )

if flag :
break
if flag :
break
for d in np . arange ( d+0 . 00001 , d-0 . 00001 ,-0 . 000001 ):
theta_min =D*np .pi/d
for theta in np . arange ( theta_min +6 , theta_min ,-0 . 1 ):
flag = judge ( theta ,d , v0 )
if flag :
break
if flag :
break
for d in np . arange ( d+0 . 000001 , d-0 . 000001 ,-0 . 0000001 ):
theta_min =D*np .pi/d
for theta in np . arange ( theta_min +6 , theta_min ,-0 . 1 ):
flag = judge ( theta ,d , v0 )
if flag :
break
if flag :
break
#细化最小的螺距
print( d ) #输出在调头空间外不会发生碰撞的最小的螺距
```

## dataproduce

```python
import numpy as np
from function import f4 #用于计算螺线上速度的斜率
from function import f5 #用于计算盘出螺线上的位置迭代
from zeropoint import zero2 #用于计算函数f5的零点
d=1 . 7 #螺距
v0=1 #龙头速度
D=9 #调头空间的直径
d0_1 =3 . 41-0 . 275 *2 #龙头板长
d0_2 =2 . 2-0 . 275 *2 #龙身和龙尾板长
theta0 =D*np .pi/d #计算龙头0时刻时的极角
x0=D*np .cos ( theta0 )/2
y0=D*np .sin ( theta0 )/2
#计算龙头0时刻的坐标（x，y）
k=f4 ( theta0 )
l1=2*np .abs ( y0-k*x0 )/np . sqrt ( k ** 2+1 )
l2=np . sqrt ( D ** 2-l1 ** 2 )
r=D ** 2/( 6*l1 ) #计算第二段圆弧的半径
x1=x0+2*r/np . sqrt ( 1+1/k ** 2 )
y1=-( x1-x0 )/k+y0
#计算第一段圆弧的圆心坐标
x2=-x0-r/np . sqrt ( 1+1/k ** 2 )
y2=-( x2+x0 )/k-y0
#计算第二段圆弧的圆心坐标
aleph =np . arccos ( l2/( r*3 ) )+np .pi/2 #计算两段圆弧的圆心角
theta_chair0_1 =np . arccos (( 8*r ** 2- d0_1 ** 2 )/( 8*r ** 2 ) )
#计算第一节龙身前把手到达第一段圆弧时龙头的位置参数theta
theta_chair0_2 =np . arccos (( 2*r ** 2- d0_1 ** 2 )/( 2*r ** 2 ) )
#计算第一节龙身前把手到达第二段圆弧时龙头的位置参数theta
a= theta0 -np .pi
b= theta0 -np .pi/2
theta_chair0_3 = zero2 ( f5 ,a ,b , 10 ** (-8 ) ,d , d0_1 , theta0 -np .pi)
#计算第一节龙身前把手到达盘出螺线时龙头的位置参数theta
theta_chair_1 =np . arccos (( 8*r ** 2- d0_2 ** 2 )/( 8*r ** 2 ) )

#计算龙身后把手到达第一段圆弧时前把手的位置参数theta
theta_chair_2 =np . arccos (( 2*r ** 2- d0_2 ** 2 )/( 2*r ** 2 ) )
#计算龙身后把手到达第二段圆弧时前把手的位置参数theta
theta_chair_3 = zero2 ( f5 ,a ,b , 10 ** (-8 ) ,d , d0_2 , theta0 -np .pi)
#计算龙身后把手到达盘出螺线时前把手的位置参数theta
t1=2*r* aleph /v0 #计算龙头到达第二段圆弧的时刻
t2=t1+r* aleph /v0 #计算龙头到达盘出螺线的时刻
theta1 =np . arctan (( y1-y0 )/( x1-x0 ) )+np .pi #计算第一段圆弧的进入点相对于圆心
的极角
theta2 =np . arctan (( y2+y0 )/( x2+x0 ) ) #计算第二段圆弧的离开点相对于圆心的极角
x_=-x0/3
y_=-y0/3
#计算两段圆弧交界点的坐标
print ( theta0 , x0 , y0 ,r , x1 , y1 , x2 , y2 , aleph , theta_chair0_1 ,
theta_chair0_2 ,
theta_chair0_3 , theta_chair_1 , theta_chair_2 , theta_chair_3 , t1 ,
t2 , theta1 ,
theta2 , x_ , y_ )
#输出各项重要参数
```

## positioniteration

```python
import numpy as np
from function import f3 #用于计算盘入螺线上的位置迭代
from function import f5 #用于计算盘出螺线上的位置迭代
from function import f6 #用于计算前把手在第一段圆弧而后把手在盘入螺线的情形
from zeropoint import zero2 #用于计算函数f3和f5的零点
from zeropoint import zero3 #用于计算函数f6的零点
def iteration1 ( theta_last , flag_last , flag_chair ):
d=1 . 7 #螺距
D=9 #调头空间的直径
theta0 =16 . 6319611 #龙头0时刻时的极角
r=1 . 5027088 #第二段圆弧的半径
aleph =3 . 0214868 #两段圆弧的圆心角
if flag_chair ==0:
d0=3 . 41-0 . 275 *2
theta_1 =0 . 9917636
theta_2 =2 . 5168977
theta_3 =14 . 1235657
else :
d0=2 . 2-0 . 275 *2
theta_1 =0 . 5561483
theta_2 =1 . 1623551
theta_3 =13 . 8544471
#确定板长和三个重要位置参数
if flag_last ==1:
theta = zero2 ( f3 , theta_last , theta_last +np .pi/2 , 10 ** (-8 ) ,d , d0 ,
theta_last )
#计算后把手的位置参数theta
flag=1 #返回后把手所在曲线的类型
#计算前把手和后把手都在盘入螺线的情形
elif flag_last ==2:
if theta_last < theta_1 :
b=np . sqrt ( 2-2*np .cos( theta_last ) )*r*2
beta =( aleph - theta_last )/2
l=np . sqrt ( b ** 2+D ** 2/4-b*D*np .cos( beta ) )
gamma =np . arcsin ( b*np .sin( beta )/l )
theta = zero3 ( f6 , theta0 , theta0 +np .pi/2 , 10 ** (-8 ) ,d , d0 ,

theta0 ,l , gamma )
#计算后把手的位置参数theta
flag=1 #返回后把手所在曲线的类型
#计算前把手在第一段圆弧而后把手在盘入螺线的情形
else :
theta = theta_last - theta_1
#计算后把手的位置参数theta
flag=2 #返回后把手所在曲线的类型
#计算前把手和后把手都在第一段圆弧的情形
elif flag_last ==3:
if theta_last < theta_2 :
a=np . sqrt ( 10-6*np .cos( theta_last ) )*r
phi=np . arccos (( 4*r ** 2+a ** 2-d0 ** 2 )/( 4*a*r ) )
beta =np . arcsin ( r*np .sin ( theta_last )/a )
theta = aleph -phi+ beta
#计算后把手的位置参数theta
flag=2 #返回后把手所在曲线的类型
#计算前把手在第二段圆弧而后把手在第一段圆弧的情形
else :
theta = theta_last - theta_2
#计算后把手的位置参数theta
flag=3 #返回后把手所在曲线的类型
#计算前把手和后把手都在第二段圆弧的情形
else :
if theta_last < theta_3 :
p=d*( theta_last +np .pi)/( 2*np .pi)
a=np . sqrt ( p ** 2+D ** 2/4-p*D*np .cos( theta_last - theta0 +np .
pi) )
beta =np . arcsin ( p*np .sin ( theta_last - theta0 +np .pi)/a )
gamma = beta -( np .pi- aleph )/2
b=np . sqrt ( a ** 2+r ** 2-2*a*r*np .cos( gamma ) )
sigma =np . arcsin ( a*np .sin( gamma )/b )
phi=np . arccos (( r ** 2+b ** 2-d0 ** 2 )/( 2*r*b ) )
theta = aleph -phi+ sigma
#计算后把手的位置参数theta
flag=3 #返回后把手所在曲线的类型
#计算前把手在盘出螺线而后把手在第二段圆弧的情形
else :
a= theta_last -np .pi/2
b= theta_last
theta = zero2 ( f5 ,a ,b , 10 ** (-8 ) ,d , d0 , theta_last )
#计算后把手的位置参数theta
flag=4 #返回后把手所在曲线的类型
#计算前把手和后把手都在盘出螺线的情形
return [theta , flag ]
#用于计算位置迭代
```

## velocityiteration

```python
import numpy as np
from function import f4 #用于计算螺线上速度的斜率
def iteration2 ( v_last , flag_last , flag , theta_last , theta , x_last , y_last
,x , y ):
x1=-0 . 7600091
y1=-1 . 3057264
#计算第一段圆弧的圆心坐标
x2=1 . 7359325
y2=2 . 4484020

#计算第二段圆弧的圆心坐标
k_chair =( y_last -y )/( x_last -x ) #计算板凳的斜率
v=-1
if flag_last ==1 and flag ==1:
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算前把手和后把手都在盘入螺线时两个速度的斜率
elif flag_last ==2 and flag ==1:
k_v_last =-( x_last -x1 )/( y_last -y1 )
k_v=f4 ( theta )
#计算前把手在第一段圆弧而后把手在盘入螺线时两个速度的斜率
elif flag_last ==2 and flag ==2:
v= v_last
#计算前把手和后把手都在第一段圆弧的情形
elif flag_last ==3 and flag ==2:
k_v_last =-( x_last -x2 )/( y_last -y2 )
k_v=-( x-x1 )/( y-y1 )
#计算前把手在第二段圆弧而后把手在第一段圆弧时两个速度的斜率
elif flag_last ==3 and flag ==3:
v= v_last
#计算前把手和后把手都在第二段圆弧的情形
elif flag_last ==4 and flag ==3:
theta_last = theta_last +np .pi
k_v_last =f4 ( theta_last )
k_v=k_v=-( x-x2 )/( y-y2 )
#计算前把手在盘出螺线而后把手在第二段圆弧时两个速度的斜率
else :
theta_last = theta_last +np .pi
theta = theta +np .pi
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算前把手和后把手都在盘出螺线时两个速度的斜率
if v==-1:
aleph1 =np . arctan ( np .abs (( k_v_last - k_chair )/( 1+ k_v_last *
k_chair ) ) )
aleph2 =np . arctan ( np .abs (( k_v - k_chair )/( 1+k_v * k_chair ) ) )
#计算两个速度与板凳的夹角
v= v_last *np .cos( aleph1 )/np .cos( aleph2 ) #计算当前把手的速度
return v
#用于计算速度迭代
import numpy as np
from function import f4 #用于计算螺线上速度的斜率
def iteration2 ( v_last , flag_last , flag , theta_last , theta , x_last , y_last
,x , y ):
x1=-0 . 7600091
y1=-1 . 3057264
#计算第一段圆弧的圆心坐标
x2=1 . 7359325
y2=2 . 4484020
#计算第二段圆弧的圆心坐标
k_chair =( y_last -y )/( x_last -x ) #计算板凳的斜率
v=-1
if flag_last ==1 and flag ==1:
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算前把手和后把手都在盘入螺线时两个速度的斜率
elif flag_last ==2 and flag ==1:
k_v_last =-( x_last -x1 )/( y_last -y1 )

k_v=f4 ( theta )
#计算前把手在第一段圆弧而后把手在盘入螺线时两个速度的斜率
elif flag_last ==2 and flag ==2:
v= v_last
#计算前把手和后把手都在第一段圆弧的情形
elif flag_last ==3 and flag ==2:
k_v_last =-( x_last -x2 )/( y_last -y2 )
k_v=-( x-x1 )/( y-y1 )
#计算前把手在第二段圆弧而后把手在第一段圆弧时两个速度的斜率
elif flag_last ==3 and flag ==3:
v= v_last
#计算前把手和后把手都在第二段圆弧的情形
elif flag_last ==4 and flag ==3:
theta_last = theta_last +np .pi
k_v_last =f4 ( theta_last )
k_v=k_v=-( x-x2 )/( y-y2 )
#计算前把手在盘出螺线而后把手在第二段圆弧时两个速度的斜率
else :
theta_last = theta_last +np .pi
theta = theta +np .pi
k_v_last =f4 ( theta_last )
k_v=f4 ( theta )
#计算前把手和后把手都在盘出螺线时两个速度的斜率
if v==-1:
aleph1 =np . arctan ( np .abs (( k_v_last - k_chair )/( 1+ k_v_last *
k_chair ) ) )
aleph2 =np . arctan ( np .abs (( k_v - k_chair )/( 1+k_v * k_chair ) ) )
#计算两个速度与板凳的夹角
v= v_last *np .cos( aleph1 )/np .cos( aleph2 ) #计算当前把手的速度
return v
于计算速度迭代
```

## 第四问代码

```python
import numpy as np
import pandas as pd
from function import f2 #用于计算龙头位置随时间的变化
from zeropoint import zero1 #用于计算函数f2的零点
from number import number #用于保留6位小数
from positioniteration import iteration1 #用于计算位置迭代
from velocityiteration import iteration2 #用于计算速度迭代
d=1 . 7 #螺距
v0=1 #龙头速度
theta0 =16 . 6319611 #龙头0时刻时的极角
r=1 . 5027088 #第二段圆弧的半径
aleph =3 . 0214868 #两段圆弧的圆心角
t1=9 . 0808299 #龙头到达第二段圆弧的时刻
t2=13 . 6212449 #龙头到达盘出螺线的时刻
x1=-0 . 7600091
y1=-1 . 3057264
#第一段圆弧的圆心坐标
x2=1 . 7359325
y2=2 . 4484020
#第二段圆弧的圆心坐标
theta1 =4 . 0055376 #第一段圆弧的进入点相对于圆心的极角
theta2 =0 . 8639449 #第二段圆弧的离开点相对于圆心的极角
lst_chair_theta =[]
lst_chair_flag =[]

for t in np . arange (-100 , 101 ):
if t<0:
theta_chair0 = zero1 ( f2 , theta0 , 100 , 10 ** (-8 ) , theta0 , v0 ,t , d )
flag_chair0 =1
elif t==0:
theta_chair0 = theta0
flag_chair0 =1
elif t<t1:
theta_chair0 =v0*t/( 2*r )
flag_chair0 =2
elif t<t2:
theta_chair0 =v0*( t-t1 )/r
flag_chair0 =3
else :
theta_chair0 = zero1 ( f2 , theta0 , 100 , 10 ** (-8 ) , theta0 , v0 ,-t+t2 , d
)-np .pi
flag_chair0 =4
#给出龙头的位置参数theta和所在曲线的类型参数flag
lst_theta =[ theta_chair0 ]
lst_flag =[ flag_chair0 ]
for i in np . arange ( 223 ):
theta_last = lst_theta [-1] #获得上一个把手的位置参数theta
flag_last = lst_flag [-1] #获得上一个把手所在曲线的类型参数flag
[theta , flag ]= iteration1 ( theta_last , flag_last , i )
lst_theta . append ( theta )
lst_flag . append ( flag )
lst_chair_theta . append ( lst_theta )
lst_chair_flag . append ( lst_flag )
#计算当前把手的位置参数theta和所在曲线的类型参数flag
lst_chair_flag =np . array ( lst_chair_flag )
lst_chair_theta =np . array ( lst_chair_theta )
lst_chair_xy =[]
for i in np . arange ( 201 ):
lst=[]
for j in np . arange ( 224 ):
flag = lst_chair_flag [i , j] #获得当前把手的位置参数theta
theta = lst_chair_theta [i , j] #获得当前把手所在曲线的类型参数flag
if flag ==1:
p=d* theta /( 2*np .pi)
x=p*np .cos( theta )
y=p*np .sin( theta )
elif flag ==2:
x=x1+2*r*np .cos( theta1 - theta )
y=y1+2*r*np .sin( theta1 - theta )
elif flag ==3:
x=x2+r*np .cos( theta2 + theta - aleph )
y=y2+r*np .sin( theta2 + theta - aleph )
else :
p=d*( theta +np .pi)/( 2*np .pi)
x=p*np .cos( theta )
y=p*np .sin( theta )
#计算当前把手的坐标（x，y）
lst . append ( x )
lst . append ( y )
lst_chair_xy . append ( lst )
lst_chair_xy =np . array ( lst_chair_xy ) . T
lst_chair_xy = number ( lst_chair_xy , 6 ) #保留6位小数
df=pd . DataFrame ( lst_chair_xy )

df . to_excel (" result4_1 . xlsx ", index = False )
#保存数据到Excel中
lst_chair_v =[]
for i in np . arange ( 201 ):
lst_v =[v0]
for j in np . arange ( 223 ):
flag_last = lst_chair_flag [i , j]
theta_last = lst_chair_theta [i , j]
flag = lst_chair_flag [i , j+1]
theta = lst_chair_theta [i , j+1]
x_last = lst_chair_xy [j*2 , i]
y_last = lst_chair_xy [j*2+1 , i]
x= lst_chair_xy [j*2+2 , i]
y= lst_chair_xy [j*2+3 , i]
#获得上一个把手和当前把手的坐标，角度参数theta和所在曲线的位置参数flag
v_last = lst_v [-1] #获得上一个把手的速度
v= iteration2 ( v_last , flag_last , flag , theta_last , theta , x_last ,
y_last ,x , y )
lst_v . append ( v )
#计算当前把手的速度
lst_chair_v . append ( lst_v )
lst_chair_v =np . array ( lst_chair_v ) . T
lst_chair_v = number ( lst_chair_v , 6 ) #保留6位小数
df=pd . DataFrame ( lst_chair_v )
df . to_excel (" result4_2 . xlsx ", index = False )
#保存数据到Excel中
```

## velctorytheta

```python
import numpy as np
from positioniteration import iteration1 #用于计算位置迭代
from velocityiteration import iteration2 #用于计算速度迭代
def f ( flag , theta ):
d=1 . 7 #螺距
r=1 . 5027088 #第二段圆弧的半径
aleph =3 . 0214868 #两段圆弧的圆心角
x1=-0 . 7600091
y1=-1 . 3057264
#第一段圆弧的圆心坐标
x2=1 . 7359325
y2=2 . 4484020
#第二段圆弧的圆心坐标
theta1=4 . 0055376 #计算第一段圆弧的进入点相对于圆心的极角
theta2=0 . 8639449 #计算第二段圆弧的离开点相对于圆心的极角
if flag ==1:
p=d* theta /( 2*np .pi)
x=p*np .cos( theta )
y=p*np .sin( theta )
#计算位于盘入螺线时的坐标
elif flag ==2:
x=x1+2*r*np .cos( theta1 - theta )
y=y1+2*r*np .sin( theta1 - theta )
#计算位于第一段圆弧时的坐标
elif flag ==3:
x=x2+r*np .cos( theta2 + theta - aleph )
y=y2+r*np .sin( theta2 + theta - aleph )
#计算位于第二段圆弧时的坐标
else :

p=d*( theta +np .pi)/( 2*np .pi)
x=p*np .cos( theta )
y=p*np .sin( theta )
#计算位于盘出螺线时的坐标
return [x , y]
#用于根据位置参数theta和所在曲线类型计算坐标
def v_theta ( theta_last , flag_last , flag_chair , v_last ):
[theta , flag ]= iteration1 ( theta_last , flag_last , flag_chair )
#计算位置参数theta和所在曲线类型
[x_last , y_last ]=f ( flag_last , theta_last )
[x , y]=f ( flag , theta )
#计算前把手和后把手坐标
v= iteration2 ( v_last , flag_last , flag , theta_last , theta , x_last ,
y_last ,x , y )
#计算速度
return [theta ,v , flag ]
#用于计算速度和位置
```

## 问题五代码

```python
import numpy as np
from velctorytheta import v_theta #用于计算速度和位置
v0=1 #龙头速度
v_max =2 #最大速度
theta0 =16 . 6319611 #龙头0时刻时的极角
theta_chair0_3 =14 . 1235657 #第一节龙身前把手到达盘出螺线时龙头的位置参数theta
lst_theta0 =np . arange ( theta0 -np .pi , theta_chair0_3 , 0 . 001 )
lst_v =[]
lst_flag =[]
lst_theta =[]
for theta0 in lst_theta0 :
[theta ,v , flag ]= v_theta ( theta0 ,4 ,0 , v0 )
lst_theta . append ( theta )
lst_v . append ( v )
lst_flag . append ( flag )
for j in np . arange ( 2 ):
for i in np . arange (len( lst_theta0 ) ):
theta_last = lst_theta [i]
v_last = lst_v [i]
flag_last = lst_flag [i]
[theta ,v , flag ]= v_theta ( theta_last , flag_last ,1 , v_last )
lst_theta [i]= theta
lst_v [i]=v
lst_flag [i]= flag
#获得该过程中第三节龙身前把手的速度随龙头位置参数theta的变化
theta0 = lst_theta0 [ lst_v . index (max ( lst_v ) )] #获得速度最大时龙头的位置参
数theta
lst_theta0 =np . arange ( theta0 -0 . 001 , theta0 +0 . 001 , 0 . 000001 )
lst_v =[]
lst_flag =[]
lst_theta =[]
for theta0 in lst_theta0 :
[theta ,v , flag ]= v_theta ( theta0 ,4 ,0 , v0 )
lst_theta . append ( theta )
lst_v . append ( v )
lst_flag . append ( flag )
for j in np . arange ( 2 ):
for i in np . arange (len( lst_theta0 ) ):

theta_last = lst_theta [i]
v_last = lst_v [i]
flag_last = lst_flag [i]
[theta ,v , flag ]= v_theta ( theta_last , flag_last ,1 , v_last )
lst_theta [i]= theta
lst_v [i]=v
lst_flag [i]= flag
#细分速度最大时龙头的位置
v0_max = v_max *v0/max ( lst_v ) #计算龙头的最大速度
print (max( lst_v ) )
print ( v0_max ) #输出龙头的最大速度
```
